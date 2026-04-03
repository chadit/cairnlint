package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestWrappedContextBackground verifies the wrappedcontextbackground analyzer flags context.WithCancel/WithTimeout/WithDeadline wrapping context.Background() in test files.
func TestWrappedContextBackground(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("wrappedcontextbackground"), "wrappedcontextbackground")
}
