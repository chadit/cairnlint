package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoRuntimeNumGoroutine verifies the noruntimenumgoroutine analyzer flags runtime.NumGoroutine() calls in test files where goleak should be used for leak detection.
func TestNoRuntimeNumGoroutine(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("noruntimenumgoroutine"), "noruntimenumgoroutine")
}
