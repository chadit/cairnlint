package typednilerror

import "fmt"

// MyError is a custom error type for testing typed nil behavior.
type MyError struct{ msg string }

func (e *MyError) Error() string { return e.msg }

// BadTypedNilReturn returns a typed nil pointer as error.
func BadTypedNilReturn() error {
	var err *MyError

	return err // want `returning typed nil as error interface produces non-nil error`
}

// BadExplicitTypedNil returns an explicit typed nil cast.
func BadExplicitTypedNil() error {
	return (*MyError)(nil) // want `returning typed nil as error interface produces non-nil error`
}

// GoodExplicitNil returns untyped nil.
func GoodExplicitNil() error {
	return nil
}

// GoodNonNilError returns a real error value.
func GoodNonNilError() error {
	return &MyError{msg: "something failed"}
}

// GoodFmtErrorf returns an fmt.Errorf result.
func GoodFmtErrorf() error {
	return fmt.Errorf("something failed")
}
