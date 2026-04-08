package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestTestCryptoInProd verifies the testcryptoinprod analyzer flags test-only
// crypto package imports in production code but allows them in test files.
func TestTestCryptoInProd(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("testcryptoinprod"), "testcryptoinprod")
}
