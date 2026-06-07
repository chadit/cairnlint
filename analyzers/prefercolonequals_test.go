package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestPreferColonEquals verifies the prefercolonequals analyzer flags var x = v with a non-zero value and leaves zero values and typed declarations alone.
func TestPreferColonEquals(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("prefercolonequals"), "prefercolonequals")
}
