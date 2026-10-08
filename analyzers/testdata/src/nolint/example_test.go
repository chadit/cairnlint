// Fixture scope note:
//
// This file is exercised by TestNolintSuppression, which runs only the
// contextbackground analyzer. other_test.go in this package puts its own
// calls on the same line numbers as the suppressed lines here, so append
// new cases at the end of this file rather than inserting them. If a future
// test runs additional analyzers against this fixture, remember:
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

// golangci-lint's wildcard name suppresses every analyzer.
func TestNolintAllWildcard(t *testing.T) {
	_ = context.Background() //nolint:all // wildcard
	_ = t
}

// golangci-lint trims whitespace after the colon and around each name.
func TestNolintWithSpaces(t *testing.T) {
	_ = context.Background() //nolint: someother, contextbackground // why
	_ = t
}

// Names are case-insensitive, as in golangci-lint.
func TestNolintUpperCase(t *testing.T) {
	_ = context.Background() //nolint:ContextBackground
	_ = t
}

// An explicit empty list is not the bare form and matches nothing.
func TestNolintEmptyListDoesNotSuppress(t *testing.T) {
	_ = context.Background() //nolint: // want `use t.Context\(\) instead of context.Background\(\) in tests`
	_ = t
}

// A directive mentioned mid-sentence is prose, not a directive, as in
// golangci-lint, which requires nolint at the start of the comment.
func TestNolintMidCommentDoesNotSuppress(t *testing.T) {
	_ = context.Background() // see //nolint docs // want `use t.Context\(\) instead of context.Background\(\) in tests`
	_ = t
}

// A hyphen after the word is not a directive boundary.
func TestNolintHyphenSuffixDoesNotSuppress(t *testing.T) {
	_ = context.Background() //nolint-ish // want `use t.Context\(\) instead of context.Background\(\) in tests`
	_ = t
}

// A single slash inside a name does not end the list; only `//` does.
func TestNolintSingleSlashInList(t *testing.T) {
	_ = context.Background() //nolint:a/b,contextbackground // explanation
	_ = t
}
