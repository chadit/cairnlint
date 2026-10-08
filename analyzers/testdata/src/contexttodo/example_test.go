package contexttodo_test

import (
	"context"
	"os"
	"testing"
	"testing/synctest"
)

func TestTODOOutsideSynctest(t *testing.T) {
	_ = context.TODO() // want `use t.Context\(\) instead of context.TODO\(\) in tests`
	_ = t
}

func TestTODOInsideSynctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_ = context.TODO()
	})
}

// TestMain receives *testing.M, which has no Context method.
func TestMain(m *testing.M) {
	_ = context.TODO()

	os.Exit(m.Run())
}

// t.Context() is already canceled when Cleanup functions run.
func TestCleanupClosure(t *testing.T) {
	t.Cleanup(func() {
		_ = context.TODO()
	})
}

// A helper in a test file stays in scope: it can take a testing.TB or a
// context from its caller.
func placeholder() context.Context {
	return context.TODO() // want `use t.Context\(\) instead of context.TODO\(\) in tests`
}
