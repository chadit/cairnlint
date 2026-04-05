// cairnlint runs custom Go analysis rules that replace ruleguard
// and grep-based checks in lint.sh.
package main

import (
	"os"

	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/chadit/cairnlint/analyzers"
)

func main() {
	all := analyzers.All()

	agentMode := consumeAgentFlag() || analyzers.DetectAgentCaller()
	if agentMode {
		all = append(all, analyzers.WrapAgentFileOutput(analyzers.AgentOnly())...)
	}

	multichecker.Main(analyzers.WrapWithNolint(analyzers.WrapSkipGenerated(all))...)
}

// consumeAgentFlag removes --agent or -agent from os.Args before
// multichecker.Main parses flags (it would reject unknown flags).
// Returns true if the flag was present.
func consumeAgentFlag() bool {
	found := false
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
