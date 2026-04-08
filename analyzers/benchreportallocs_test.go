package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestBenchReportAllocs verifies the benchreportallocs analyzer flags
// benchmark functions that are missing b.ReportAllocs().
func TestBenchReportAllocs(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("benchreportallocs"), "benchreportallocs")
}
