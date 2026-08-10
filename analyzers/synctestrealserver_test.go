package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestSynctestRealServer verifies the synctestrealserver analyzer flags real httptest servers inside a synctest bubble and leaves outside uses alone.
func TestSynctestRealServer(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("synctestrealserver"), "synctestrealserver")
}
