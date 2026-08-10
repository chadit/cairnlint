package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestSynctestSleepWait verifies the synctestsleepwait analyzer flags time.Sleep immediately followed by synctest.Wait and skips separated calls.
func TestSynctestSleepWait(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("synctestsleepwait"), "synctestsleepwait")
}
