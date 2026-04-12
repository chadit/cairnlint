// cairnlint runs custom Go analysis rules that replace ruleguard
// and grep-based checks in lint.sh.
package main

import (
	"fmt"
	"os"

	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/chadit/cairnlint/analyzers"
)

func main() {
	if consumeListFlag() {
		if err := analyzers.PrintLinters(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "cairnlint: print linters: %v\n", err)
			os.Exit(1)
		}

		return
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
