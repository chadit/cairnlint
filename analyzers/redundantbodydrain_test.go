package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestRedundantBodyDrain verifies the redundantbodydrain analyzer flags discarded response body drains and leaves copies whose result is used alone.
func TestRedundantBodyDrain(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("redundantbodydrain"), "redundantbodydrain")
}
