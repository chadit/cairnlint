package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoDefaultHTTPClient verifies the nodefaulthttpclient analyzer flags direct references to http.DefaultClient which has no timeout configured.
func TestNoDefaultHTTPClient(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("nodefaulthttpclient"), "nodefaulthttpclient")
}
