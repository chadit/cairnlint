package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoExportTest verifies the noexporttest analyzer flags export_test.go files that expose package internals instead of testing through the public API.
func TestNoExportTest(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("noexporttest"), "noexporttest")
}
