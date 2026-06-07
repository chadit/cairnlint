package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestConstMixedCaps verifies the constmixedcaps analyzer flags const names with an underscore or k-prefix.
func TestConstMixedCaps(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("constmixedcaps"), "constmixedcaps")
}
