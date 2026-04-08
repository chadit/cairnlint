package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestChanDirection verifies the chandirection analyzer flags bidirectional
// channel parameters where a directional type (<-chan or chan<-) should be used.
func TestChanDirection(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("chandirection"), "chandirection")
}
