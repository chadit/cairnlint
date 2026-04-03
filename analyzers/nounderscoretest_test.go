package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoUnderscoreTest verifies the nounderscoretest analyzer flags Test/Benchmark/Fuzz function names containing underscores where MixedCaps should be used.
func TestNoUnderscoreTest(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("nounderscoretest"), "nounderscoretest")
}
