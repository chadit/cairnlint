package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chadit/cairnlint/analyzers"
)

// TestNolintSuppression verifies that WrapWithNolint honors //nolint
// directives in all the forms golangci-lint supports: trailing on the
// same line, leading above a statement, bare (all analyzers), unrelated
// (no effect), and leading above a whole function.
func TestNolintSuppression(t *testing.T) {
	t.Parallel()

	wrapped := analyzers.WrapWithNolint([]*analysis.Analyzer{findAnalyzer("contextbackground")})

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, wrapped[0], "nolint")
}
