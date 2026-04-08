package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestGoWGGo verifies the gowggo analyzer flags go wg.Go(...) wrapping
// that races WaitGroup.Add with Wait.
func TestGoWGGo(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("gowggo"), "gowggo")
}
