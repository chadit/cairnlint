package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestPreferBLoop verifies the preferbloop analyzer flags old-style b.N benchmark loops and suggests b.Loop() instead (Go 1.24+).
func TestPreferBLoop(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("preferbloop"), "preferbloop")
}
