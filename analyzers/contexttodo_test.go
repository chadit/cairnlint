package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestContextTODO verifies the contexttodo analyzer flags context.TODO() calls in test files where t.Context() should be used instead.
func TestContextTODO(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("contexttodo"), "contexttodo")
}
