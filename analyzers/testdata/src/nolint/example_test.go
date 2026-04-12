// Fixture scope note:
//
// This file is exercised by TestNolintSuppression, which runs only the
// contextbackground analyzer. If a future test runs additional analyzers
// against this fixture, remember:
//   - commentedcode skips *_test.go files, so this file is safe from it.
//     A non-test fixture added alongside would NOT be safe — prose like
//     "see nolint docs" is fine, but lines starting with Go keywords
//     (`// var x = ...`, `// return err`) would trip it.
//   - Any new analyzer added to the test harness needs its own suppression
//     fixtures that match the forms tested here.
package nolint_test

import (
	"context"
	"testing"
)

func TestTrailingNolint(t *testing.T) {
	_ = context.Background() //nolint:contextbackground // suppressed on same line
	_ = t
}

func TestLeadingNolint(t *testing.T) {
	//nolint:contextbackground // directive above the statement
	_ = context.Background()
	_ = t
}

func TestBareNolintSuppressesAll(t *testing.T) {
	_ = context.Background() //nolint
	_ = t
}

func TestUnrelatedNolintDoesNotSuppress(t *testing.T) {
	_ = context.Background() //nolint:someotheranalyzer // want `use t.Context\(\) instead of context.Background\(\) in tests`
	_ = t
}

//nolint:contextbackground // whole-function suppression above a FuncDecl
func TestFunctionLevelNolint(t *testing.T) {
	_ = context.Background()
	_ = context.Background()
	_ = t
}

func TestNotSuppressed(t *testing.T) {
	_ = context.Background() // want `use t.Context\(\) instead of context.Background\(\) in tests`
	_ = t
}

// Identifier-like tokens must not trigger a bare nolint match.
func TestNolintAsSubstring(t *testing.T) {
	_ = context.Background() //nolintfoo // want `use t.Context\(\) instead of context.Background\(\) in tests`
	_ = t
}

// Prose containing the word nolint must not suppress.
func TestNolintInProseDoesNotSuppress(t *testing.T) {
	// see nolint docs for details on how suppressions work
	_ = context.Background() // want `use t.Context\(\) instead of context.Background\(\) in tests`
	_ = t
}

// Trailing junk after the name list must not silently expand the list.
func TestNolintWithTrailingJunk(t *testing.T) {
	_ = context.Background() //nolint:someother//contextbackground // want `use t.Context\(\) instead of context.Background\(\) in tests`
	_ = t
}
