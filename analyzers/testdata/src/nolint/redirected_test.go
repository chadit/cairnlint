// Fixture scope note:
//
// The //line directive below moves every later position to moved_test.go,
// a file that does not exist. Diagnostics and directives are both keyed by
// the adjusted filename, so a //nolint here must still suppress its line.
package nolint_test

import (
	"context"
	"testing"
)

//line moved_test.go:20:1
func TestNolintFollowsLineDirective(t *testing.T) {
	_ = context.Background() //nolint:contextbackground // suppressed under the redirected position
	_ = t
}

func TestReportedUnderLineDirective(t *testing.T) {
	_ = context.Background() // want `use t.Context\(\) instead of context.Background\(\) in tests`
	_ = t
}
