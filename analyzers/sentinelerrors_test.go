package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestSentinelErrors verifies the sentinelerrors analyzer flags sentinel error declarations (var ErrFoo = errors.New(...)) in files not named errors.go.
func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("sentinelerrors"), "sentinelerrors")
}
