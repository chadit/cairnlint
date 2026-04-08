package testcryptoinprod

import (
	_ "crypto/mlkem/mlkemtest" // want `test-only crypto package crypto/mlkem/mlkemtest must not be imported in production code`
	_ "testing/cryptotest"     // want `test-only crypto package testing/cryptotest must not be imported in production code`
)
