package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestDocTutorialVoice verifies the doctutorialvoice analyzer flags tutorial
// and instructional voice phrases in doc comments while leaving declarative
// doc comments alone.
func TestDocTutorialVoice(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("doctutorialvoice"), "doctutorialvoice")
}
