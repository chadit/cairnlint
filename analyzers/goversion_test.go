package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestGoVersionGate verifies that an analyzer recommending a standard library
// API stays quiet in files whose language version predates that API, and still
// fires in files that can use it. The fixture puts both cases in one package so
// the per-file version, not the package version, is what decides.
func TestGoVersionGate(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("prefererrorsastype"), "goversiongate")
}
