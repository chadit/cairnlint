package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoContextInStruct verifies the nocontextinstruct analyzer flags context.Context stored as a struct field where it should be passed as a function parameter instead.
func TestNoContextInStruct(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("nocontextinstruct"), "nocontextinstruct")
}
