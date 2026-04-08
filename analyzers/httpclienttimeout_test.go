package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestHTTPClientTimeout verifies the httpclienttimeout analyzer flags
// http.Client composite literals that omit the Timeout field.
func TestHTTPClientTimeout(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("httpclienttimeout"), "httpclienttimeout")
}
