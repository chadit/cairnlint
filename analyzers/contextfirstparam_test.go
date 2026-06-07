package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestContextFirstParam verifies the contextfirstparam analyzer flags a context.Context that is not first and exempts test helpers led by a testing value.
func TestContextFirstParam(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("contextfirstparam"), "contextfirstparam")
}
