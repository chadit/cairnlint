package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestBufferPeekStore verifies the bufferpeekstore analyzer flags Peek results
// used after the buffer is mutated, which causes silent data corruption since
// the returned slice aliases internal buffer memory.
func TestBufferPeekStore(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("bufferpeekstore"), "bufferpeekstore")
}
