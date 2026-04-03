package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestSynctestSleep verifies the synctestsleep analyzer flags time.Sleep calls in test files outside synctest.Test closures.
func TestSynctestSleep(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("synctestsleep"), "synctestsleep")
}
