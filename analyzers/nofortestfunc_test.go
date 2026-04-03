package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoForTestFunc verifies the nofortestfunc analyzer flags functions with ForTest or ForTesting suffixes that expose internals for test use.
func TestNoForTestFunc(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("nofortestfunc"), "nofortestfunc")
}
