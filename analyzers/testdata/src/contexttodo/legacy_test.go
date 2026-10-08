//go:build go1.23

package contexttodo_test

import (
	"context"
	"testing"
)

// t.Context() arrived in Go 1.24, so a file pinned below it gets no suggestion.
func TestTODOBelowGo124(t *testing.T) {
	_ = context.TODO()
	_ = t
}
