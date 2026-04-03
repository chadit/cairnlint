package wrappedcontextbackground_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

func TestWithCancelBackground(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background()) // want `use t.Context\(\) as base context in tests, not context.Background\(\)`
	defer cancel()
	_ = ctx
}

func TestWithTimeoutBackground(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second) // want `use t.Context\(\) as base context in tests, not context.Background\(\)`
	defer cancel()
	_ = ctx
}

func TestWithDeadlineBackground(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second)) // want `use t.Context\(\) as base context in tests, not context.Background\(\)`
	defer cancel()
	_ = ctx
}

// Not flagged: wrapping t.Context() is fine.
func TestWithCancelTContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	_ = ctx
}

// Not flagged: inside synctest.
func TestWithCancelInsideSynctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_ = ctx
	})
}
