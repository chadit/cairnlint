package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoInlineMocks verifies the noinlinemocks analyzer flags mock struct type declarations in test files outside test/mocks/ where they should be centralized.
func TestNoInlineMocks(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("noinlinemocks"), "noinlinemocks")
}
