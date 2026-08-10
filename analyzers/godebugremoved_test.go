package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestGoDebugRemoved verifies the godebugremoved analyzer flags go:debug directives naming settings removed in Go 1.27 and leaves live settings alone.
func TestGoDebugRemoved(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("godebugremoved"), "godebugremoved")
}
