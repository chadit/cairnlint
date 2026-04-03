package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestCommentedCode verifies the commentedcode analyzer flags comments in non-test Go files that look like disabled code starting with Go keywords.
func TestCommentedCode(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("commentedcode"), "commentedcode")
}
