package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestDBQueryWithBareBackground verifies the dbquerywithbarebackground analyzer flags QueryContext/ExecContext/QueryRowContext calls that pass context.Background() instead of a request-scoped context.
func TestDBQueryWithBareBackground(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("dbquerywithbarebackground"), "dbquerywithbarebackground")
}
