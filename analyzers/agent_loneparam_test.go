package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAgentLoneParam verifies the agent-only analyzer flags a parameter fixed
// across every call site, and stays quiet for a parameter that varies, a
// callee with one call site, a callee passed around as a value, and an
// exported callee whose other call sites are invisible.
func TestAgentLoneParam(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAgentAnalyzer("agentloneparam"), "agentloneparam")
}
