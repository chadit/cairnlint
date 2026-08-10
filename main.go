// cairnlint runs custom Go analysis rules that replace ruleguard
// and grep-based checks in lint.sh.
package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/chadit/cairnlint/analyzers"
)

// exitUsage is the status returned for a rejected command line, kept distinct
// from the exit code the multichecker uses to signal findings.
const exitUsage = 2

func main() {
	if consumeListFlag() {
		if err := analyzers.PrintLinters(os.Stdout); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "cairnlint: print linters: %v\n", err)

			os.Exit(1)
		}

		return
	}

	// Resolve build tags before multichecker parses flags, because the
	// multichecker's own -tags flag is a deprecated no-op shim (see
	// golang.org/x/tools go/analysis/internal/analysisflags/flags.go).
	// Packages gated by //go:build <tag> would otherwise look empty to
	// go/packages and trigger "matched no packages".
	tagsFlag := ConsumeTagsFlag()

	// -tags=auto triggers the multi-pass auto-discovery runner before
	// the normal single-pass flow takes over. The runner re-execs this
	// binary once per discovered user tag (plus a default-build pass),
	// so every lint rule fires against every build configuration without
	// callers having to enumerate tags in their lint.sh.
	if tagsFlag == AutoTagsSentinel {
		exe, err := os.Executable()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "cairnlint: resolve executable: %v\n", err)

			os.Exit(1)
		}

		os.Exit(RunAutoTagsPasses(context.Background(), os.Stdout, os.Stderr, exe, os.Args[1:]))
	}

	if tagsFlag != "" {
		if err := PropagateBuildTags(tagsFlag); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "cairnlint: propagate -tags: %v\n", err)

			os.Exit(1)
		}
	}

	if flag, rejected := rejectedEditFlag(); rejected {
		_, _ = fmt.Fprintf(os.Stderr,
			"cairnlint: %s is not supported; cairnlint reports findings and never edits source\n", flag)

		os.Exit(exitUsage)
	}

	all := analyzers.All()

	agentMode := consumeAgentFlag() || analyzers.DetectAgentCaller()
	if agentMode {
		all = append(all, analyzers.WrapAgentFileOutput(analyzers.AgentOnly())...)
	}

	// All and AgentOnly already strip suggested fixes, so nothing reaching the
	// multichecker carries an edit it could apply.
	multichecker.Main(analyzers.WrapWithNolint(analyzers.WrapSkipGenerated(all))...)
}

// editFlags are the multichecker driver flags that rewrite source files.
// The driver registers them for every analysis binary, so cairnlint has to
// turn them away rather than decline to define them.
var editFlags = []string{"-fix", "--fix", "-diff", "--diff"} //nolint:gochecknoglobals // package-internal lookup table

// rejectedEditFlag reports the first source-editing flag present in os.Args.
//
// WrapWithoutFixes already leaves the driver with nothing to apply, so this
// check exists to fail loudly rather than accept -fix and quietly change
// nothing, which would read as "the fixes were already applied".
func rejectedEditFlag() (string, bool) {
	for _, arg := range os.Args[1:] {
		name, _, _ := strings.Cut(arg, "=")

		if slices.Contains(editFlags, name) {
			return name, true
		}
	}

	return "", false
}

// consumeListFlag removes --list/--linters (and single-dash forms) from
// os.Args before multichecker.Main parses flags. Returns true if any form
// was present so the caller knows to print the linter catalog and exit.
// Using a flag rather than a bare subcommand avoids colliding with user
// packages that happen to be named "list" or "linters".
func consumeListFlag() bool {
	var found bool

	filtered := os.Args[:0]

	for _, arg := range os.Args {
		switch arg {
		case "--list", "-list", "--linters", "-linters":
			found = true

			continue
		}

		filtered = append(filtered, arg)
	}

	os.Args = filtered

	return found
}

// consumeAgentFlag removes --agent or -agent from os.Args before
// multichecker.Main parses flags (it would reject unknown flags).
// Returns true if the flag was present.
func consumeAgentFlag() bool {
	var found bool

	filtered := os.Args[:0]

	for _, arg := range os.Args {
		if arg == "--agent" || arg == "-agent" {
			found = true

			continue
		}

		filtered = append(filtered, arg)
	}

	os.Args = filtered

	return found
}

// ConsumeTagsFlag removes -tags=<value>, --tags=<value>, -tags <value>, and
// --tags <value> from os.Args, returning the last value seen or "" if none.
// Extraction happens before multichecker parses flags because its own -tags
// flag is a deprecated no-op shim that swallows the value without effect.
func ConsumeTagsFlag() string {
	var tags string

	filtered := make([]string, 0, len(os.Args))

	var idx int
	for idx < len(os.Args) {
		arg := os.Args[idx]

		switch {
		case strings.HasPrefix(arg, "--tags="):
			tags = strings.TrimPrefix(arg, "--tags=")
		case strings.HasPrefix(arg, "-tags="):
			tags = strings.TrimPrefix(arg, "-tags=")
		case arg == "--tags" || arg == "-tags":
			if idx+1 < len(os.Args) {
				tags = os.Args[idx+1]
				idx++
			}
		default:
			filtered = append(filtered, arg)
		}

		idx++
	}

	os.Args = filtered

	return tags
}

// PropagateBuildTags prepends -tags=<tags> to the GOFLAGS environment
// variable so go/packages, which shells out to `go list`, sees them during
// package loading. Prepending rather than replacing preserves any other
// flags the caller already put in GOFLAGS.
func PropagateBuildTags(tags string) error {
	existing := os.Getenv("GOFLAGS")

	combined := "-tags=" + tags
	if existing != "" {
		combined = combined + " " + existing
	}

	if err := os.Setenv("GOFLAGS", combined); err != nil {
		return fmt.Errorf("set GOFLAGS: %w", err)
	}

	return nil
}
