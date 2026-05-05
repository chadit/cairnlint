package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestDiscardedContext verifies the discardedcontext analyzer flags function parameters declared as _ context.Context that break the cancellation chain.
func TestDiscardedContext(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("discardedcontext"), "discardedcontext", "discardedcontextmocks/mocks")
}
