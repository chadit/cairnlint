package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAgentAIBuzzwords verifies the aibuzzwords agent-only analyzer flags
// AI-flavored vocabulary, hedging, formal transitions, clichés, academic
// setups, and preachy universals in comment text.
func TestAgentAIBuzzwords(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAgentAnalyzer("aibuzzwords"), "aibuzzwords")
}
