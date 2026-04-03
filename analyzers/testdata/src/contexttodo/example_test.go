package contexttodo_test

import (
	"context"
	"testing"
	"testing/synctest"
)

func TestTODOOutsideSynctest(t *testing.T) {
	ctx := context.TODO() // want `use t.Context\(\) instead of context.TODO\(\) in tests`
	_ = ctx
}

func TestTODOInsideSynctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.TODO()
		_ = ctx
	})
}
