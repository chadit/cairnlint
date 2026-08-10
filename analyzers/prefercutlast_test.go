package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestPreferCutLast verifies the prefercutlast analyzer flags LastIndex results used as slice bounds and leaves other LastIndex uses alone.
func TestPreferCutLast(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("prefercutlast"), "prefercutlast")
}
