package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAgentStubBody verifies the agent-only analyzer flags empty bodies under a
// descriptive doc, lone canned returns, and "not implemented" sentinels, while
// leaving real one-line accessors, constructors, and intentional no-ops alone.
func TestAgentStubBody(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAgentAnalyzer("agentstubbody"), "agentstubbody")
}
