package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestReflectDeepEqualScalar verifies the reflectdeepequalscalar analyzer flags
// reflect.DeepEqual on scalar operands, reports differing concrete types as
// always false, and leaves interface, type-parameter, and composite operands alone.
func TestReflectDeepEqualScalar(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("reflectdeepequalscalar"), "reflectdeepequalscalar")
}
