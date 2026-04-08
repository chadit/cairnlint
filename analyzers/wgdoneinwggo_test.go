package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestWGDoneInWGGo verifies the wgdoneinwggo analyzer flags redundant
// wg.Done() calls inside wg.Go() closures that double-decrement the counter.
func TestWGDoneInWGGo(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("wgdoneinwggo"), "wgdoneinwggo")
}
