package main_test

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"

	cairnlint "github.com/chadit/cairnlint"
)

const (
	integrationTag  = "integration"
	wildcardPattern = "./..."
)

// TestDiscoverBuildTagsReturnsUserTags writes a minimal tree of .go files
// with a mix of user-defined tags, GOOS/GOARCH values, pseudo-tags, and
// negations, then confirms only the user-defined tags survive filtering.
func TestDiscoverBuildTagsReturnsUserTags(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	writeFile(t, root, "a.go", "//go:build integration\n\npackage a\n")
	writeFile(t, root, "sub/b.go", "//go:build e2e && !slow\n\npackage sub\n")
	writeFile(t, root, "sub/c.go", "//go:build linux || darwin\n\npackage sub\n")
	writeFile(t, root, "sub/d.go", "//go:build cgo\n\npackage sub\n")
	writeFile(t, root, "sub/e.go", "//go:build go1.26\n\npackage sub\n")
	writeFile(t, root, "sub/f.go", "//go:build custom || integration\n\npackage sub\n")
	writeFile(t, root, "plain.go", "package root\n")

	// Directories the walker must skip.
	writeFile(t, root, "testdata/skip.go", "//go:build skipped\n\npackage skip\n")
	writeFile(t, root, "vendor/skip.go", "//go:build vendored\n\npackage skip\n")
	writeFile(t, root, ".hidden/skip.go", "//go:build hidden\n\npackage skip\n")

	got, err := cairnlint.DiscoverBuildTags([]string{root})
	if err != nil {
		t.Fatalf("DiscoverBuildTags: %v", err)
	}

	want := []string{"custom", "e2e", integrationTag, "slow"}

	sort.Strings(got)

	if !equalStringSlices(got, want) {
		t.Errorf("tags: got %v, want %v", got, want)
	}
}

// TestDiscoverBuildTagsParsesLegacyPlusBuild confirms // +build (the
// pre-1.17 constraint syntax) is also recognized, since old repos can
// still have files carrying it.
func TestDiscoverBuildTagsParsesLegacyPlusBuild(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	writeFile(t, root, "legacy.go", "// +build legacytag\n\npackage legacy\n")

	got, err := cairnlint.DiscoverBuildTags([]string{root})
	if err != nil {
		t.Fatalf("DiscoverBuildTags: %v", err)
	}

	if !equalStringSlices(got, []string{"legacytag"}) {
		t.Errorf("tags: got %v, want [legacytag]", got)
	}
}

// TestDiscoverBuildTagsIgnoresNonGoFiles confirms the walker limits
// parsing to .go files so README or config files with // +build in them
// do not leak false tags into the set.
func TestDiscoverBuildTagsIgnoresNonGoFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	writeFile(t, root, "notes.md", "//go:build fakeone\n\nsome text\n")
	writeFile(t, root, "real.go", "//go:build realone\n\npackage p\n")

	got, err := cairnlint.DiscoverBuildTags([]string{root})
	if err != nil {
		t.Fatalf("DiscoverBuildTags: %v", err)
	}

	if !equalStringSlices(got, []string{"realone"}) {
		t.Errorf("tags: got %v, want [realone]", got)
	}
}

// TestDiscoverBuildTagsHandlesTrailingWildcard confirms the "./..." and
// "/abs/path/..." patterns get stripped down to their root dir so the
// walker does not try to match "..." against a filesystem entry.
func TestDiscoverBuildTagsHandlesTrailingWildcard(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	writeFile(t, root, "pkg/a.go", "//go:build wildtag\n\npackage pkg\n")

	got, err := cairnlint.DiscoverBuildTags([]string{root + "/..."})
	if err != nil {
		t.Fatalf("DiscoverBuildTags: %v", err)
	}

	if !equalStringSlices(got, []string{"wildtag"}) {
		t.Errorf("tags: got %v, want [wildtag]", got)
	}
}

// TestRunAutoTagsPassesMergesOutput runs the multi-pass driver against
// /bin/echo to confirm duplicate lines collapse and per-pass headers
// still print. Using a stub executable sidesteps the need to rebuild
// cairnlint itself for the test.
//
//nolint:paralleltest // calls t.Chdir, which forbids parallel
func TestRunAutoTagsPassesMergesOutput(t *testing.T) {
	if _, err := os.Stat("/bin/echo"); err != nil {
		t.Skip("/bin/echo unavailable on this platform")
	}

	root := t.TempDir()
	writeFile(t, root, "a.go", "//go:build alpha\n\npackage a\n")
	writeFile(t, root, "b.go", "//go:build beta\n\npackage a\n")

	t.Chdir(root)

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	// echo will ignore any -tags=<value> prefix and print the rest.
	// Pass a fixed pattern arg so the discovery walk finds our fixtures.
	code := cairnlint.RunAutoTagsPasses(t.Context(), &stdout, &stderr, "/bin/echo", []string{wildcardPattern})
	if code != 0 {
		t.Errorf("exit code: got %d, want 0", code)
	}

	// Each pass prints one echo line. Tags should appear when present.
	stdoutText := stdout.String()

	if !bytes.Contains(stdout.Bytes(), []byte(wildcardPattern)) {
		t.Errorf("stdout missing pattern echo: %q", stdoutText)
	}

	stderrText := stderr.String()

	if !bytes.Contains(stderr.Bytes(), []byte("pass default build")) {
		t.Errorf("stderr missing default-build header: %q", stderrText)
	}

	if !bytes.Contains(stderr.Bytes(), []byte("pass -tags=alpha")) {
		t.Errorf("stderr missing alpha header: %q", stderrText)
	}

	if !bytes.Contains(stderr.Bytes(), []byte("pass -tags=beta")) {
		t.Errorf("stderr missing beta header: %q", stderrText)
	}
}

// writeFile creates a file at root/relPath, making parent directories
// as needed. Keeps fixture setup declarative in each test.
func writeFile(t *testing.T, root, relPath, contents string) {
	t.Helper()

	full := filepath.Join(root, relPath)

	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdirall %s: %v", filepath.Dir(full), err)
	}

	if err := os.WriteFile(full, []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

// equalStringSlices reports whether two string slices have the same
// contents in the same order.
func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	for idx, val := range left {
		if val != right[idx] {
			return false
		}
	}

	return true
}
