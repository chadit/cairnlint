package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestPreferWGGoFanout verifies the preferwggofanout analyzer flags
// wg.Add(N) + loop-of-N-goroutines patterns and skips loops where the Add
// count and loop count cannot be reliably matched.
func TestPreferWGGoFanout(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("preferwggofanout"), "preferwggofanout")
}
