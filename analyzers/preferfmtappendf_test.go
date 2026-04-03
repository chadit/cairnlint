package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestPreferFmtAppendf verifies the preferfmtappendf analyzer flags []byte(fmt.Sprintf(...)) conversions where fmt.Appendf avoids the intermediate string allocation.
func TestPreferFmtAppendf(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("preferfmtappendf"), "preferfmtappendf")
}
