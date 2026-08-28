package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAgentLoneImpl verifies the agent-only analyzer flags an interface with a
// single implementing type in the package, while leaving interfaces with two
// implementations, no implementation, and no method set alone.
func TestAgentLoneImpl(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAgentAnalyzer("agentloneimpl"), "agentloneimpl")
}
