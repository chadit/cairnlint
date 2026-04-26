package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestDocParamBlock verifies the docparamblock analyzer flags Javadoc-style
// Parameters: and Returns: section headers in non-test function doc comments.
func TestDocParamBlock(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("docparamblock"), "docparamblock")
}
