package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestSignalHandling verifies the signalhandling analyzer flags main() functions
// that start HTTP servers or network listeners without signal handling for
// graceful shutdown.
func TestSignalHandling(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analyzer := findAnalyzer("signalhandling")

	// main() with http.ListenAndServe but no signal handling should be flagged.
	analysistest.Run(t, testdata, analyzer, "signalhandling_bad")

	// main() with signal.NotifyContext before starting a server is fine.
	analysistest.Run(t, testdata, analyzer, "signalhandling_good")
}
