package main

import (
	"bufio"
	"fmt"
	"go/build/constraint"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// AutoTagsSentinel is the -tags value that triggers discovery and a
// multi-pass lint across every user-defined build tag found in the tree.
// Exported so lint.sh and other drivers can reference the constant name.
const AutoTagsSentinel = "auto"

// reservedBuildTags lists identifiers that appear in //go:build lines but
// are NOT user-defined tags. GOOS/GOARCH values are already selected by the
// toolchain based on environment; forcing them via -tags would not enable
// cross-platform files (the go/build constraint system checks GOOS/GOARCH
// independently) but would pollute the discovered set. Compiler and
// feature identifiers (cgo, race, ...) likewise come from build mode, not
// from -tags. "ignore" is a conventional marker for files intentionally
// excluded from every build.
var reservedBuildTags = map[string]struct{}{ //nolint:gochecknoglobals // immutable lookup table, not runtime state
	// GOOS values recognized by go/build as of Go 1.22.
	"aix": {}, "android": {}, "darwin": {}, "dragonfly": {},
	"freebsd": {}, "hurd": {}, "illumos": {}, "ios": {},
	"js": {}, "linux": {}, "netbsd": {}, "openbsd": {},
	"plan9": {}, "solaris": {}, "wasip1": {}, "windows": {}, "zos": {},

	// GOARCH values recognized by go/build as of Go 1.22.
	"386": {}, "amd64": {}, "amd64p32": {}, "arm": {},
	"armbe": {}, "arm64": {}, "arm64be": {}, "loong64": {},
	"mips": {}, "mips64": {}, "mips64le": {}, "mips64p32": {},
	"mips64p32le": {}, "mipsle": {}, "ppc": {}, "ppc64": {},
	"ppc64le": {}, "riscv": {}, "riscv64": {}, "s390": {},
	"s390x": {}, "sparc": {}, "sparc64": {}, "wasm": {},

	// Meta GOOS groups and compiler/feature identifiers.
	"unix": {}, "cgo": {}, "race": {}, "msan": {}, "asan": {},
	"gc": {}, "gccgo": {}, "purego": {}, "ignore": {},
}

// DiscoverBuildTags walks the source tree rooted at patterns and returns
// the sorted set of user-defined build tags referenced by //go:build (or
// legacy // +build) constraints. Reserved identifiers (GOOS, GOARCH,
// compiler pseudo-tags, go1.* version tags) are filtered out so the
// caller can trust every returned value is worth a separate lint pass.
func DiscoverBuildTags(patterns []string) ([]string, error) {
	roots, err := patternRoots(patterns)
	if err != nil {
		return nil, err
	}

	collected := make(map[string]struct{})

	for _, root := range roots {
		if err := walkForBuildTags(root, collected); err != nil {
			return nil, err
		}
	}

	tags := make([]string, 0, len(collected))
	for tag := range collected {
		tags = append(tags, tag)
	}

	sort.Strings(tags)

	return tags, nil
}

// patternRoots converts go/packages-style patterns into filesystem roots.
// Only the filesystem-style patterns (./..., ./path, ./path/...) are
// resolved locally. Module-path patterns (example.com/foo) are outside
// the scope of auto-discovery; the caller falls back to a default-build
// pass in that case, which still reports every untagged diagnostic.
func patternRoots(patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("resolve cwd for tag discovery: %w", err)
		}

		return []string{cwd}, nil
	}

	seen := make(map[string]struct{}, len(patterns))
	roots := make([]string, 0, len(patterns))

	for _, pattern := range patterns {
		root, ok := filesystemRoot(pattern)
		if !ok {
			continue
		}

		if _, dup := seen[root]; dup {
			continue
		}

		seen[root] = struct{}{}
		roots = append(roots, root)
	}

	return roots, nil
}

// filesystemRoot strips a trailing /... wildcard and returns the directory
// the walk should start from. Returns false for patterns that are not
// filesystem paths (e.g. module import paths) so discovery can skip them
// cleanly rather than walking the wrong tree.
func filesystemRoot(pattern string) (string, bool) {
	trimmed := strings.TrimSuffix(pattern, "/...")
	if trimmed == "..." {
		trimmed = "."
	}

	if trimmed == "" {
		trimmed = "."
	}

	// Filesystem-style patterns start with ., .., /, or are a bare "."
	// Module paths contain neither slash prefix nor leading dot.
	if !strings.HasPrefix(trimmed, ".") && !filepath.IsAbs(trimmed) {
		return "", false
	}

	return trimmed, true
}

// walkForBuildTags descends rootDir, parsing build constraints from every
// .go file. Skips directories the Go toolchain ignores (dot-prefixed,
// underscore-prefixed, testdata, vendor) so the discovered tag set
// matches what a normal build would actually compile. File I/O is scoped
// to rootDir via os.Root so a symlink escape under the tree cannot cause
// reads outside the intended subtree.
func walkForBuildTags(rootDir string, dst map[string]struct{}) error {
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return fmt.Errorf("open root %s: %w", rootDir, err)
	}

	defer func() { _ = root.Close() }()

	fsys := root.FS()

	walkErr := fs.WalkDir(fsys, ".", func(path string, dirent fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dirent.IsDir() {
			if shouldSkipDir(path, dirent.Name()) {
				return filepath.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		return extractBuildTagsFromFile(root, path, dst)
	})
	if walkErr != nil {
		return fmt.Errorf("walk %s: %w", rootDir, walkErr)
	}

	return nil
}

// shouldSkipDir reports whether the walker should skip the directory.
// The walk root itself (path ".") is always entered, even when its leaf
// name would otherwise match a skip rule (e.g. cairnlint run from
// testdata/), so the user's explicit target wins over the skip list.
func shouldSkipDir(path, name string) bool {
	if path == "." {
		return false
	}

	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
		return true
	}

	switch name {
	case "testdata", "vendor", "node_modules":
		return true
	}

	return false
}

// extractBuildTagsFromFile reads the leading comment block of relPath
// (interpreted within root) and parses any //go:build or // +build line
// it finds. Stops at the first non-comment, non-whitespace line because
// build constraints must appear before the package clause.
func extractBuildTagsFromFile(root *os.Root, relPath string, dst map[string]struct{}) error {
	file, err := root.Open(relPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", relPath, err)
	}

	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		if constraint.IsGoBuild(trimmed) || constraint.IsPlusBuild(trimmed) {
			expr, parseErr := constraint.Parse(trimmed)
			if parseErr != nil {
				continue
			}

			collectTagIdentifiers(expr, dst)

			continue
		}

		// A line that is neither blank nor a //-comment nor a /*-comment
		// ends the build-constraint region. Build constraints cannot
		// appear after the package clause, so further scanning is wasted.
		if !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") {
			break
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return fmt.Errorf("scan %s: %w", relPath, scanErr)
	}

	return nil
}

// collectTagIdentifiers walks a parsed constraint expression and inserts
// any user-defined tag name into dst. Negations are unwrapped because a
// !tag constraint still tells us the project cares about `tag`.
func collectTagIdentifiers(expr constraint.Expr, dst map[string]struct{}) {
	switch node := expr.(type) {
	case *constraint.TagExpr:
		if isUserBuildTag(node.Tag) {
			dst[node.Tag] = struct{}{}
		}
	case *constraint.NotExpr:
		collectTagIdentifiers(node.X, dst)
	case *constraint.AndExpr:
		collectTagIdentifiers(node.X, dst)
		collectTagIdentifiers(node.Y, dst)
	case *constraint.OrExpr:
		collectTagIdentifiers(node.X, dst)
		collectTagIdentifiers(node.Y, dst)
	}
}

// isUserBuildTag reports whether tag is a user-defined build tag rather
// than a GOOS/GOARCH value, compiler pseudo-tag, or go1.N version gate.
// Filtering these out keeps the auto-discovery signal strictly to tags
// that make sense to enable via -tags on the command line.
func isUserBuildTag(tag string) bool {
	if _, reserved := reservedBuildTags[tag]; reserved {
		return false
	}

	// Go version identifiers (go1.22, go1.23, ...) are satisfied by the
	// toolchain version, not by -tags.
	if strings.HasPrefix(tag, "go1.") {
		return false
	}

	return true
}
