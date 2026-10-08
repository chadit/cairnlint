//go:build go1.23

package wrappedcontextbackground_test

import (
	"context"
	"testing"
)

// t.Context() arrived in Go 1.24, so a file pinned below it gets no suggestion.
func TestWithCancelBelowGo124(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = t
}
