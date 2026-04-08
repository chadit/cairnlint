package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestBuilderGrow verifies the buildergrow analyzer flags strings.Builder write
// methods inside loops that lack a preceding Grow call.
func TestBuilderGrow(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("buildergrow"), "buildergrow")
}
