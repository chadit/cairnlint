package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestMapPrealloc verifies the mapprealloc analyzer flags maps created without
// a capacity hint that are then populated inside a range loop.
func TestMapPrealloc(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("mapprealloc"), "mapprealloc")
}
