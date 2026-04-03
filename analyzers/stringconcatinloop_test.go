package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestStringConcatInLoop verifies the stringconcatinloop analyzer flags string += concatenation inside for loops where strings.Builder should be used instead.
func TestStringConcatInLoop(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("stringconcatinloop"), "stringconcatinloop")
}
