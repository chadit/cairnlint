package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestBenchResetTimer verifies the benchresettimer analyzer flags benchmark
// functions with setup code but no b.ResetTimer() call.
func TestBenchResetTimer(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("benchresettimer"), "benchresettimer")
}
