package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestPreferFatalErrGate verifies the preferfatalerrgate analyzer flags a
// non-halting t.Error in an errors.Is/As/AsType assertion when the error is
// used after the check, and leaves the documented exceptions alone.
func TestPreferFatalErrGate(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("preferfatalerrgate"), "preferfatalerrgate")
}
