package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoElse verifies the noelse analyzer flags if-else blocks where early returns or guard clauses should be used instead.
func TestNoElse(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("noelse"), "noelse")
}
