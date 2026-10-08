// Fixture scope note:
//
// This file shares a package with example_test.go so both land in one
// analysis pass. Each context.Background() below sits on the same line
// number as a suppressed call in example_test.go: 22 (trailing directive),
// 28 (directive on the line above), and 44 (inside a function-level
// span). Padding is comment text, never blank lines, because gofmt
// collapses blank runs and would shift the calls off those lines.
package nolint_test

import (
	"context"
	"testing"
)

// TestOtherFileLine22 lands its call on line 22. In example_test.go that
// line carries a trailing //nolint:contextbackground. Suppression keyed
// by line number alone silenced this call too, even though no directive
// in this file mentions it, which is the bug the (filename, line) key
// fixes.
func TestOtherFileLine22(t *testing.T) {
	_ = context.Background() // want `use t.Context\(\) instead of context.Background\(\) in tests`
	_ = t
}

// TestOtherFileLine28 lands on the line below a leading directive.
func TestOtherFileLine28(t *testing.T) {
	_ = context.Background() // want `use t.Context\(\) instead of context.Background\(\) in tests`
	_ = t
}

// The function-level directive in example_test.go attaches to the whole
// FuncDecl, so its span covers several lines, not just one. A line-only
// key spread that span across every file in the package, which made it
// the widest of the three leaks: any diagnostic in any other file whose
// line number fell inside the span disappeared, with nothing on that
// line to explain why.

// TestOtherFileLine44 lands inside that span. Under the old key the call
// here was silenced by a directive it never saw; under the (filename,
// line) key it must be reported. The `want` on this line is what proves
// the leak is closed.
func TestOtherFileLine44(t *testing.T) {
	_ = context.Background() // want `use t.Context\(\) instead of context.Background\(\) in tests`
	_ = t
}
