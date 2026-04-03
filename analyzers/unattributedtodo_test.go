package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestUnattributedTODO verifies the unattributedtodo analyzer flags TODO/FIXME/HACK/XXX comments that lack an owner name or ticket reference.
func TestUnattributedTODO(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("unattributedtodo"), "unattributedtodo")
}
