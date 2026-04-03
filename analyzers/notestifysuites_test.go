package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoTestifySuites verifies the notestifysuites analyzer flags struct types embedding suite.Suite from testify in test files.
func TestNoTestifySuites(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("notestifysuites"), "notestifysuites")
}
