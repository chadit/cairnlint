package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestPreferFmtAppendf verifies the preferfmtappendf analyzer flags []byte(fmt.Sprintf(...)) only when it feeds a Write call or an append spread, and leaves the bare conversion alone.
func TestPreferFmtAppendf(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("preferfmtappendf"), "preferfmtappendf")
}
