//go:build go1.23

package contextbackground_test

import (
	"context"
	"testing"
)

// t.Context() arrived in Go 1.24, so a file pinned below it gets no
// suggestion it cannot follow.
func TestBackgroundBelowGo124(t *testing.T) {
	_ = context.Background()
	_ = t
}
