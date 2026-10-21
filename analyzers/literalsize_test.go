package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chadit/cairnlint/analyzers"
)

// TestLiteralSize verifies the literalsize analyzer flags bare literal array
// lengths and channel capacities everywhere, and literal slice and map make
// sizes in test files. The run wraps the analyzer the way main.go does, so
// the generated fixture must stay silent.
func TestLiteralSize(t *testing.T) {
	t.Parallel()

	wrapped := analyzers.WrapSkipGenerated([]*analysis.Analyzer{findAnalyzer("literalsize")})

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, wrapped[0], "literalsize")
}
