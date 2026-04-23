package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestPreferWGGo verifies the preferwggo analyzer flags the classic
// pre-Go-1.25 pattern wg.Add(1) + go func(){ defer wg.Done(); ... }() and
// suggests wg.Go(fn).
func TestPreferWGGo(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("preferwggo"), "preferwggo")
}
