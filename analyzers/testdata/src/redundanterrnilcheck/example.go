package redundanterrnilcheck

import "errors"

// ErrSentinel is a sentinel error used by the test cases for errors.Is.
var ErrSentinel = errors.New("sentinel")

// MyErr is a concrete error type used by the errors.As/AsType test cases.
type MyErr struct {
	msg string
}

// Error satisfies the error interface for MyErr.
func (e *MyErr) Error() string {
	return e.msg
}

// doThing returns an error for the test cases to inspect.
func doThing() error {
	return ErrSentinel
}
