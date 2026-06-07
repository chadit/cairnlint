package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestMixedReceiver verifies the mixedreceiver analyzer flags a type that mixes value and pointer receivers.
func TestMixedReceiver(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("mixedreceiver"), "mixedreceiver")
}
