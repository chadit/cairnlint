package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestStmtNoClose verifies the stmtnoclose analyzer flags db.Prepare and db.PrepareContext calls without a corresponding defer Close.
func TestStmtNoClose(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("stmtnoclose"), "stmtnoclose")
}
