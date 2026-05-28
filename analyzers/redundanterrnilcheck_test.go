package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestRedundantErrNilCheck verifies the redundanterrnilcheck analyzer flags an
// err == nil test assertion when a following errors.Is/As/AsType on the same
// err already fails for nil, and leaves the documented exceptions alone.
func TestRedundantErrNilCheck(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("redundanterrnilcheck"), "redundanterrnilcheck")
}
