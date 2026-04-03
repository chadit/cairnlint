package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestQueryInLoop verifies the queryinloop analyzer flags database query calls (Query, QueryRow, Exec and their Context variants) inside for loops as N+1 patterns.
func TestQueryInLoop(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("queryinloop"), "queryinloop")
}
