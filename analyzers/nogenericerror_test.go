package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoGenericError verifies the nogenericerror analyzer flags errors.New() calls with vague messages like "error" or "failed" that lack debugging context.
func TestNoGenericError(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("nogenericerror"), "nogenericerror")
}
