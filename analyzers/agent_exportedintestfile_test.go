package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAgentExportedInTestFile verifies the agent-only analyzer flags exported
// declarations in augmented test files (same-package _test.go) while allowing
// framework functions like TestXxx, BenchmarkXxx, and TestMain.
func TestAgentExportedInTestFile(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAgentAnalyzer("agentexportedintestfile"), "agentexportedintestfile")
}
