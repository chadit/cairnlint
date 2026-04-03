package noerrstrcontains

import (
	"errors"
	"strings"
)

// Flagged: matching error strings with strings.Contains.
func checkErrBad() {
	err := errors.New("something happened")
	if strings.Contains(err.Error(), "something") { // want `do not match error strings with Contains`
		_ = err
	}
}

// Not flagged: strings.Contains on a non-error string.
func checkStringOK() {
	msg := "hello world"
	if strings.Contains(msg, "hello") {
		_ = msg
	}
}

// Not flagged: using errors.Is (the correct approach).
func checkErrGood() {
	err := errors.New("something happened")
	if errors.Is(err, errors.ErrUnsupported) {
		_ = err
	}
}
