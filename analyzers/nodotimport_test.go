package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoDotImport verifies the nodotimport analyzer flags dot imports.
func TestNoDotImport(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("nodotimport"), "nodotimport")
}
