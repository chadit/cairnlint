package nopanicinlib_test

import "testing"

// Not flagged: panic in test files is acceptable.
func TestPanicInTest(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()

	panic("test panic")
}
