package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAgentPassthrough verifies the agent-only analyzer reports a one-field
// type whose whole method set forwards once at the type, reports a lone
// forwarder on a working type at the method, and leaves methods that branch or
// reshape their arguments alone.
func TestAgentPassthrough(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAgentAnalyzer("agentpassthrough"), "agentpassthrough")
}
