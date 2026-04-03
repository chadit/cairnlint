package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestExternalTestPkg verifies the externaltestpkg analyzer flags test files using internal test packages instead of the external _test package suffix.
func TestExternalTestPkg(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("externaltestpkg"), "externaltestpkg")
}
