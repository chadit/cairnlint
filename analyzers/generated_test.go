package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chadit/cairnlint/analyzers"
)

// TestSkipGenerated verifies that WrapSkipGenerated drops diagnostics only
// in files with a real generated-code header before the package clause. A
// file that mentions the marker in a later comment, and a preprocessed file
// whose //line directive points back at a hand-written source, both keep
// their diagnostics.
func TestSkipGenerated(t *testing.T) {
	t.Parallel()

	wrapped := analyzers.WrapSkipGenerated([]*analysis.Analyzer{findAnalyzer("emdash")})

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, wrapped[0], "generated")
}
