package preferfatalerrgate

import (
	"errors"
	"testing"
)

// TestIsErrUsedAfter is flagged: err is used after a non-halting errors.Is gate.
func TestIsErrUsedAfter(t *testing.T) {
	err := doThing()
	if !errors.Is(err, ErrSentinel) {
		t.Errorf("want sentinel") // want `use t\.Fatal`
	}

	_ = err.Error()
}

// TestAsTargetUsedAfter is flagged: the As target is used after the gate.
func TestAsTargetUsedAfter(t *testing.T) {
	err := doThing()

	var target *MyErr
	if !errors.As(err, &target) {
		t.Errorf("want MyErr") // want `use t\.Fatal`
	}

	_ = target.msg
}

// TestAsTypeErrUsedAfter is flagged: err is used after an errors.AsType gate.
func TestAsTypeErrUsedAfter(t *testing.T) {
	err := doThing()
	if _, ok := errors.AsType[*MyErr](err); !ok {
		t.Errorf("want MyErr") // want `use t\.Fatal`
	}

	_ = err.Error()
}

// TestIsNothingAfter is not flagged: nothing uses err after the check, so a
// non-halting failure is fine.
func TestIsNothingAfter(t *testing.T) {
	err := doThing()
	if !errors.Is(err, ErrSentinel) {
		t.Errorf("want sentinel")
	}
}

// TestIsAlreadyFatal is not flagged: the failure branch already halts.
func TestIsAlreadyFatal(t *testing.T) {
	err := doThing()
	if !errors.Is(err, ErrSentinel) {
		t.Fatalf("want sentinel")
	}

	_ = err.Error()
}

// TestIsErrorThenReturn is not flagged: the failure branch returns, so code
// after the check only runs on success.
func TestIsErrorThenReturn(t *testing.T) {
	err := doThing()
	if !errors.Is(err, ErrSentinel) {
		t.Errorf("want sentinel")

		return
	}

	_ = err.Error()
}

// TestIsInGoroutine is not flagged: t.Fatal is illegal off the test goroutine.
func TestIsInGoroutine(t *testing.T) {
	done := make(chan struct{})

	go func() {
		err := doThing()
		if !errors.Is(err, ErrSentinel) {
			t.Errorf("want sentinel")
		}

		_ = err.Error()
		close(done)
	}()

	<-done
}

// TestIsOtherUsedAfter is not flagged: the later use is a different variable.
func TestIsOtherUsedAfter(t *testing.T) {
	err := doThing()
	other := doThing()
	if !errors.Is(err, ErrSentinel) {
		t.Errorf("want sentinel")
	}

	_ = other.Error()
}
