package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestConsistentReceiverName verifies the consistentreceivername analyzer flags receiver names that differ across a type's methods.
func TestConsistentReceiverName(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("consistentreceivername"), "consistentreceivername")
}
