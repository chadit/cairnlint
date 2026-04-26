package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestEmdash verifies the emdash analyzer flags Unicode em dash characters
// (U+2014) in comments and leaves hyphens and en dashes alone.
func TestEmdash(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("emdash"), "emdash")
}
