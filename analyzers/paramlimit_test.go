package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestParamLimit verifies the paramlimit analyzer flags signatures with more than 3 domain parameters, leaves exempt plumbing uncounted, and skips mocks directories.
func TestParamLimit(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("paramlimit"), "paramlimit", "paramlimitmocks/mocks")
}
