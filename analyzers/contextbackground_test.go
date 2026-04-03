package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestContextBackground verifies the contextbackground analyzer flags context.Background() calls in test files where t.Context() should be used instead.
func TestContextBackground(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("contextbackground"), "contextbackground")
}
