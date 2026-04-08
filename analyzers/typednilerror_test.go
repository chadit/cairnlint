package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestTypedNilError verifies the typednilerror analyzer flags returning a typed
// nil pointer as an error interface. This produces a non-nil interface value,
// causing `if err != nil` checks to unexpectedly succeed.
func TestTypedNilError(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("typednilerror"), "typednilerror")
}
