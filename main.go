// cairnlint runs custom Go analysis rules that replace ruleguard
// and grep-based checks in lint.sh.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/chadit/cairnlint/analyzers"
)

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

	all := analyzers.All()

	agentMode := consumeAgentFlag() || analyzers.DetectAgentCaller()
	if agentMode {
		all = append(all, analyzers.WrapAgentFileOutput(analyzers.AgentOnly())...)
	}

	multichecker.Main(analyzers.WrapWithNolint(analyzers.WrapSkipGenerated(all))...)
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
