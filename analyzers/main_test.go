package analyzers_test

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain verifies no goroutines leak across the test suite.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
