package redundanterrnilcheck

import (
	"errors"
	"testing"
)

// TestRedundantIs flags a nil check made redundant by a following errors.Is.
func TestRedundantIs(t *testing.T) {
	err := doThing()
	if err == nil { // want `redundant err == nil check`
		t.Fatal("expected error")
	}

	if !errors.Is(err, ErrSentinel) {
		t.Errorf("want sentinel: %v", err)
	}
}

// TestRedundantAs flags a nil check made redundant by a following errors.As.
func TestRedundantAs(t *testing.T) {
	err := doThing()

	var target *MyErr
	if err == nil { // want `redundant err == nil check`
		t.Fatalf("expected error")
	}

	if !errors.As(err, &target) {
		t.Errorf("want MyErr")
	}
}

// TestRedundantAsType flags a nil check made redundant by errors.AsType.
func TestRedundantAsType(t *testing.T) {
	err := doThing()
	if err == nil { // want `redundant err == nil check`
		t.Fatal("expected error")
	}

	if _, ok := errors.AsType[*MyErr](err); !ok {
		t.Errorf("want MyErr")
	}
}

// TestRedundantReversed flags `nil == err` followed by errors.Is.
func TestRedundantReversed(t *testing.T) {
	err := doThing()
	if nil == err { // want `redundant err == nil check`
		t.Error("expected error")
	}

	if !errors.Is(err, ErrSentinel) {
		t.Errorf("want sentinel")
	}
}

// TestIsNilTarget is not flagged: errors.Is(err, nil) is itself the nil check.
func TestIsNilTarget(t *testing.T) {
	err := doThing()
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, nil) {
		t.Errorf("want non-nil error")
	}
}

// TestDifferentVar is not flagged: the errors.Is runs on a different variable.
func TestDifferentVar(t *testing.T) {
	err := doThing()
	other := doThing()
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(other, ErrSentinel) {
		t.Errorf("want sentinel")
	}
}

// TestNotAdjacent is not flagged: a statement separates the two checks.
func TestNotAdjacent(t *testing.T) {
	err := doThing()
	if err == nil {
		t.Fatal("expected error")
	}

	_ = err

	if !errors.Is(err, ErrSentinel) {
		t.Errorf("want sentinel")
	}
}

// TestPlainAs is not flagged: no preceding nil check.
func TestPlainAs(t *testing.T) {
	err := doThing()

	var target *MyErr
	if !errors.As(err, &target) {
		t.Errorf("want MyErr")
	}
}

// helperReturns is not flagged: the nil branch does real work, not a test
// failure, so it is a production-style happy path rather than an assertion.
func helperReturns() error {
	err := doThing()
	if err == nil {
		return nil
	}

	if !errors.Is(err, ErrSentinel) {
		return err
	}

	return nil
}
