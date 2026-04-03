package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoAAAComments verifies the noaaacomments analyzer flags // Arrange, // Act, and // Assert section marker comments in test files.
func TestNoAAAComments(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("noaaacomments"), "noaaacomments")
}
