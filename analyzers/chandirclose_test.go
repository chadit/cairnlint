package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestChanDirClose verifies the chandirclose analyzer flags close() calls on
// bidirectional channel parameters where the function may not own the channel.
func TestChanDirClose(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("chandirclose"), "chandirclose")
}
