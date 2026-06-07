package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestPreferAny verifies the preferany analyzer flags the empty interface{} type and leaves any and non-empty interfaces alone.
func TestPreferAny(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("preferany"), "preferany")
}
