package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoPanicInLib verifies the nopanicinlib analyzer flags panic() calls in non-test files where errors should be returned instead.
func TestNoPanicInLib(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("nopanicinlib"), "nopanicinlib")
}
