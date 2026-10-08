package wrappedcontextbackground_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

func TestWithCancelBackground(t *testing.T) {
	_, cancel := context.WithCancel(context.Background()) // want `use t.Context\(\) as base context in tests, not context.Background\(\)`
	defer cancel()
}

func TestWithTimeoutBackground(t *testing.T) {
	_, cancel := context.WithTimeout(context.Background(), time.Second) // want `use t.Context\(\) as base context in tests, not context.Background\(\)`
	defer cancel()
}

func TestWithDeadlineBackground(t *testing.T) {
	_, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second)) // want `use t.Context\(\) as base context in tests, not context.Background\(\)`
	defer cancel()
}

// The Cause variants and AfterFunc take the same base context.
func TestCauseAndAfterFuncForms(t *testing.T) {
	_, cancel := context.WithCancelCause(context.Background()) // want `use t.Context\(\) as base context in tests, not context.Background\(\)`
	defer cancel(errors.New("done"))

	_, cancelTimeout := context.WithTimeoutCause(context.Background(), time.Second, errors.New("slow")) // want `use t.Context\(\) as base context in tests, not context.Background\(\)`
	defer cancelTimeout()

	_, cancelDeadline := context.WithDeadlineCause(context.Background(), time.Now().Add(time.Second), errors.New("late")) // want `use t.Context\(\) as base context in tests, not context.Background\(\)`
	defer cancelDeadline()

	_ = context.AfterFunc(context.Background(), func() {}) // want `use t.Context\(\) as base context in tests, not context.Background\(\)`
}

// Not flagged here: WithoutCancel is never canceled whatever its base, so
// this analyzer leaves it to contextbackground, which reports the inner call.
func TestWithoutCancel(t *testing.T) {
	_ = context.WithoutCancel(context.Background())
}

// Not flagged: wrapping t.Context() is fine.
func TestWithCancelTContext(t *testing.T) {
	_, cancel := context.WithCancel(t.Context())
	defer cancel()
}

// Not flagged: inside synctest.
func TestWithCancelInsideSynctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		defer cancel()
	})
}

// Not flagged: t.Context() is already canceled when Cleanup runs, so a
// Cleanup that needs a live deadline has to start from Background.
func TestWithTimeoutInCleanup(t *testing.T) {
	t.Cleanup(func() {
		_, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
	})
}

// A helper in a test file stays in scope: it can take a testing.TB or a
// context from its caller.
func newCancelable() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background()) // want `use t.Context\(\) as base context in tests, not context.Background\(\)`
}
