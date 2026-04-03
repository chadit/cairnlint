package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestPreferVarZero verifies the prefervarzero analyzer flags short declarations with zero-value literals like s := "" or n := 0 where var declarations make zero-value intent explicit.
func TestPreferVarZero(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("prefervarzero"), "prefervarzero")
}
