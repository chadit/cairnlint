package testcryptoinprod

import (
	"testing"

	_ "crypto/mlkem/mlkemtest"
	_ "testing/cryptotest"
)

// TestGood verifies test files can import test-only crypto packages.
func TestGood(t *testing.T) {}
