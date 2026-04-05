package analyzers_test

import (
	"golang.org/x/tools/go/analysis"

	"github.com/chadit/cairnlint/analyzers"
)

// findAnalyzer returns the analyzer with the given name from the full
// cairnlint registry. Panics if no match, surfacing test typos immediately.
func findAnalyzer(name string) *analysis.Analyzer {
	for _, a := range analyzers.All() {
		if a.Name == name {
			return a
		}
	}

	panic("analyzer not found: " + name)
}

// findAgentAnalyzer returns the analyzer with the given name from the
// agent-only registry. Panics if no match.
func findAgentAnalyzer(name string) *analysis.Analyzer {
	for _, a := range analyzers.AgentOnly() {
		if a.Name == name {
			return a
		}
	}

	panic("agent analyzer not found: " + name)
}
