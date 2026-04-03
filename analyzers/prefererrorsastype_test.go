package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestPreferErrorsAsType verifies the prefererrorsastype analyzer flags errors.As() calls and suggests errors.AsType[T]() for type-safe error unwrapping (Go 1.26+).
func TestPreferErrorsAsType(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("prefererrorsastype"), "prefererrorsastype")
}
