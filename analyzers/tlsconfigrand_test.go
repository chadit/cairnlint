package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestTLSConfigRand verifies the tlsconfigrand analyzer flags writes to the deprecated tls.Config.Rand field and leaves same-named fields alone.
func TestTLSConfigRand(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("tlsconfigrand"), "tlsconfigrand")
}
