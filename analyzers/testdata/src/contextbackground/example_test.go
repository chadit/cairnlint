package contextbackground_test

import (
	"context"
	"testing"
	"testing/synctest"
)

func TestBackgroundOutsideSynctest(t *testing.T) {
	ctx := context.Background() // want `use t.Context\(\) instead of context.Background\(\) in tests`
	_ = ctx
}

func TestBackgroundInsideSynctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		_ = ctx
	})
}
