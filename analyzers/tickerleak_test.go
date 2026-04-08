package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestTickerLeak verifies the tickerleak analyzer flags time.NewTicker and time.NewTimer calls without a corresponding defer Stop.
func TestTickerLeak(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("tickerleak"), "tickerleak")
}
