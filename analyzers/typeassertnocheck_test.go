package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestTypeAssertNoCheck verifies the typeassertnocheck analyzer flags single-value type assertions that panic on failure instead of using the comma-ok form.
func TestTypeAssertNoCheck(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("typeassertnocheck"), "typeassertnocheck")
}
