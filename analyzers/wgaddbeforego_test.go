package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestWGAddBeforeGo verifies the wgaddbeforego analyzer flags redundant
// wg.Add calls immediately before wg.Go, which double-counts the WaitGroup.
func TestWGAddBeforeGo(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("wgaddbeforego"), "wgaddbeforego")
}
