package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestTestStructuredBlock verifies the teststructuredblock analyzer flags
// the structured section headers (Workflow, Test Environment, Expected
// Behavior, Purpose, Simulates) in test function doc comments.
func TestTestStructuredBlock(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("teststructuredblock"), "teststructuredblock")
}
