package contextbackground_test

import (
	"context"
	"os"
	"testing"
	"testing/synctest"
	"time"
)

func TestBackgroundOutsideSynctest(t *testing.T) {
	_ = context.Background() // want `use t.Context\(\) instead of context.Background\(\) in tests`
	_ = t
}

func TestBackgroundInsideSynctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_ = context.Background()
	})
}

// TestMain receives *testing.M, which has no Context method, so there is
// nothing to suggest.
func TestMain(m *testing.M) {
	_ = context.Background()

	os.Exit(m.Run())
}

// An Example has no testing parameter at all.
func Example() {
	_ = context.Background()
}

// A helper in a test file stays in scope: it can take a testing.TB or a
// context from its caller.
func newBackground() context.Context {
	return context.Background() // want `use t.Context\(\) instead of context.Background\(\) in tests`
}

// A closure inside a test captures t, so the suggestion still applies.
func TestClosureInsideTest(t *testing.T) {
	done := make(chan struct{})

	go func() {
		_ = context.Background() // want `use t.Context\(\) instead of context.Background\(\) in tests`

		close(done)
	}()

	<-done
	_ = newBackground
}

// A defer runs before the test returns, while t.Context() is still live.
func TestDeferStillLive(t *testing.T) {
	defer func() {
		_ = context.Background() // want `use t.Context\(\) instead of context.Background\(\) in tests`
	}()

	_ = t
}

// t.Context() is canceled just before Cleanup functions run, so a Cleanup
// body that needs a live context cannot use it.
func TestCleanupClosure(t *testing.T) {
	t.Cleanup(func() {
		_ = context.Background()
	})
}

// b.Cleanup and f.Cleanup get the same exemption.
func BenchmarkCleanupClosure(b *testing.B) {
	b.Cleanup(func() {
		_ = context.Background()
	})

	_ = context.Background() // want `use t.Context\(\) instead of context.Background\(\) in tests`
}

func FuzzCleanupClosure(f *testing.F) {
	f.Cleanup(func() {
		_ = context.Background()
	})

	_ = context.Background() // want `use t.Context\(\) instead of context.Background\(\) in tests`
}

// testing.TB carries Cleanup and Context too.
func helperWithTB(tb testing.TB) {
	tb.Cleanup(func() {
		_ = context.Background()
	})

	_ = context.Background() // want `use t.Context\(\) instead of context.Background\(\) in tests`
}

func TestHelperWithTB(t *testing.T) {
	helperWithTB(t)
}

// An alias of testing.T is seen through. The helper cannot be named Test*
// because go test rejects that signature.
type aliasT = testing.T

func aliasCleanup(t *aliasT) {
	t.Cleanup(func() {
		_ = context.Background()
	})
}

func TestAliasCleanup(t *testing.T) {
	aliasCleanup(t)
}

// A closure bound to a local and passed to Cleanup later runs as a Cleanup
// body all the same.
func TestCleanupViaLocal(t *testing.T) {
	stop := func() {
		_ = context.Background()
	}

	t.Cleanup(stop)
}

// A function passed to Cleanup by name, and everything it calls, runs on
// the Cleanup path.
func teardown() {
	_ = context.Background()

	closeConn()
}

func closeConn() {
	_ = context.Background()
}

func TestCleanupByName(t *testing.T) {
	t.Cleanup(teardown)
}

// A helper called from inside a Cleanup closure runs there too.
func cleanupDB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_ = ctx
	_ = t
}

func TestCleanupCallsHelper(t *testing.T) {
	t.Cleanup(func() {
		cleanupDB(t)
	})
}

// A Background() that is the base of a context.With* call is reported by
// wrappedcontextbackground instead, so the line carries one diagnostic.
func TestWrappedLeftToSibling(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = context.AfterFunc(context.Background(), func() {})
}

// WithoutCancel is not in the sibling's list, so its base is reported here.
func TestWithoutCancelStillReported(t *testing.T) {
	_ = context.WithoutCancel(context.Background()) // want `use t.Context\(\) instead of context.Background\(\) in tests`
}
