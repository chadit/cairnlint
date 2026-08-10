package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestURLClone verifies the urlclone analyzer flags shallow copies of url.Values and url.URL and leaves unrelated copies alone.
func TestURLClone(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("urlclone"), "urlclone")
}
