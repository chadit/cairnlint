package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestReflectNoKindCheck verifies the reflectnokindcheck analyzer flags
// reflect Fields/NumField calls that lack a preceding Kind guard.
func TestReflectNoKindCheck(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("reflectnokindcheck"), "reflectnokindcheck")
}
