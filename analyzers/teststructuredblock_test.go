package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestTestStructuredBlock verifies the teststructuredblock analyzer reports
// the listed section headers in test doc comments, in bold and lowercase
// forms, and leaves code-block samples and TestMain alone.
func TestTestStructuredBlock(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("teststructuredblock"), "teststructuredblock")
}
