package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestTestHelperMarker verifies the testhelper analyzer flags helpers taking *testing.T that skip t.Helper() and exempts entry points and blank receivers.
func TestTestHelperMarker(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("testhelper"), "testhelper")
}
