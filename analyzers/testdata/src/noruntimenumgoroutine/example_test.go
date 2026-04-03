package noruntimenumgoroutine_test

import (
	"runtime"
	"testing"
	"testing/synctest"
)

// Flagged: runtime.NumGoroutine outside synctest.
func TestGoroutineCountOutside(t *testing.T) {
	n := runtime.NumGoroutine() // want `runtime\.NumGoroutine\(\) is unreliable for leak detection; use goleak instead`
	_ = n
}

// Not flagged: inside synctest.Test closure.
func TestGoroutineCountInsideSynctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := runtime.NumGoroutine()
		_ = n
	})
}

// Not flagged: non-test file calls are not checked by this analyzer.
