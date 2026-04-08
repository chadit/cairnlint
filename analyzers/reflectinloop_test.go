package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestReflectInLoop verifies the reflectinloop analyzer flags reflect.ValueOf and reflect.TypeOf calls inside loop bodies.
func TestReflectInLoop(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("reflectinloop"), "reflectinloop")
}
