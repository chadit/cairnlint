package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestPoolResetBeforePut verifies the poolresetbeforeput analyzer flags
// sync.Pool.Put calls where the object hasn't been reset first. Objects
// returned to a pool without cleanup carry stale data into the next Get.
func TestPoolResetBeforePut(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, findAnalyzer("poolresetbeforeput"), "poolresetbeforeput")
}
