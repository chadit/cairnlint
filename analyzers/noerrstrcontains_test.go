package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoErrStrContains verifies the noerrstrcontains analyzer flags string-matching on error messages via strings.Contains(err.Error(), ...) and testify equivalents.
func TestNoErrStrContains(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("noerrstrcontains"), "noerrstrcontains")
}
