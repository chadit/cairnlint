package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestSQLInjection verifies the sqlinjection analyzer flags fmt.Sprintf calls whose format string contains SQL keywords indicating injection risk.
func TestSQLInjection(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("sqlinjection"), "sqlinjection")
}
