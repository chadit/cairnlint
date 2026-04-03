package nogenericerror

import "errors"

// Flagged: vague error messages.
func badErrors() {
	_ = errors.New("error")            // want `error message "error" is too vague`
	_ = errors.New("failed")           // want `error message "failed" is too vague`
	_ = errors.New("operation failed") // want `error message "operation failed" is too vague`
	_ = errors.New("invalid")          // want `error message "invalid" is too vague`
	_ = errors.New("invalid input")    // want `error message "invalid input" is too vague`
	_ = errors.New("unknown error")    // want `error message "unknown error" is too vague`
	_ = errors.New("internal error")   // want `error message "internal error" is too vague`
}

// Not flagged: descriptive error messages.
func goodErrors() {
	_ = errors.New("failed to connect to database on port 5432")
	_ = errors.New("user ID must be a positive integer")
	_ = errors.New("config file not found at /etc/app/config.yaml")
}

// Not flagged: dynamic error message (not a string literal).
func dynamicError(msg string) error {
	return errors.New(msg)
}
