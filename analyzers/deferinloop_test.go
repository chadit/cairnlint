package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestDeferInLoop verifies the deferinloop analyzer flags defer statements inside for loop bodies where deferred calls stack until the function returns.
func TestDeferInLoop(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("deferinloop"), "deferinloop")
}
