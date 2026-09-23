package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNamedResultUse verifies the namedresultuse analyzer flags named results
// used as variables while leaving documentary names and defer assignment alone.
func TestNamedResultUse(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("namedresultuse"), "namedresultuse")
}
