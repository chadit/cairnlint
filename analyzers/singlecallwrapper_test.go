package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestSingleCallWrapper verifies the singlecallwrapper analyzer flags an
// unexported function, in a test file too, whose body only returns another
// package's function called with its parameters in order and fixed arguments,
// and leaves every other shape the doc comment names alone.
func TestSingleCallWrapper(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("singlecallwrapper"), "singlecallwrapper")
}
