package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestSelfReceiver verifies the selfreceiver analyzer flags receivers named this, self, or me.
func TestSelfReceiver(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("selfreceiver"), "selfreceiver")
}
