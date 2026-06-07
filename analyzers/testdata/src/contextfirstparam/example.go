package contextfirstparam

import (
	"context"
	"testing"
)

// Flagged: context after another parameter.
func Bad(x int, ctx context.Context) {} // want `context.Context should be the first parameter`

// Not flagged: context first.
func Good(ctx context.Context, x int) {}

// Not flagged: no context parameter.
func NoCtx(x int) {}

// Not flagged: a leading testing value takes precedence in helpers.
func Helper(t *testing.T, ctx context.Context) {}
