package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestTestUnderscorePrefix verifies the testunderscoreprefix analyzer flags Test names with an underscore right after the prefix (Test_Foo) while allowing TestFoo_Bar, Benchmark_, and Fuzz_ grouping.
func TestTestUnderscorePrefix(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("testunderscoreprefix"), "testunderscoreprefix")
}
