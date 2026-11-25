package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestGenericPackageName verifies the genericpackagename analyzer flags vague
// package names like util and leaves a specific name such as netutil alone.
func TestGenericPackageName(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("genericpackagename"), "genericpackagename", "genericpackagename/netutil")
}
