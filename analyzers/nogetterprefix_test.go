package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoGetterPrefix verifies the nogetterprefix analyzer flags Get/get prefixes on zero-arg accessors.
func TestNoGetterPrefix(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("nogetterprefix"), "nogetterprefix")
}
