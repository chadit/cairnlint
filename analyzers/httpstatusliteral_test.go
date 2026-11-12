package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chadit/cairnlint/analyzers"
)

// TestHTTPStatusLiteral verifies the httpstatusliteral analyzer flags a named
// status literal written to or compared with a field or variable that reaches
// a response status, one marked in another package included. Both fixture
// packages are roots, so their exported marks are checked too. The run wraps
// the analyzer the way main.go does, so the generated fixture stays silent.
func TestHTTPStatusLiteral(t *testing.T) {
	t.Parallel()

	wrapped := analyzers.WrapSkipGenerated([]*analysis.Analyzer{findAnalyzer("httpstatusliteral")})

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, wrapped[0], "httpstatusliteral", "httpstatusliteral/cloudtest")
}
