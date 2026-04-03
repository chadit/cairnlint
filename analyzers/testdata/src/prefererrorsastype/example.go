package prefererrorsastype

import (
	"errors"
	"os"
)

// Flagged: errors.As should use errors.AsType[T] in Go 1.26+.
func checkPathError(err error) {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) { // want `use errors\.AsType\[T\]\(err\) instead of errors\.As`
		_ = pathErr
	}
}

// Not flagged: errors.Is is fine.
func checkSentinel(err error) {
	if errors.Is(err, os.ErrNotExist) {
		_ = err
	}
}

// Not flagged: errors.AsType is the preferred form (if it existed in testdata).
// We cannot call errors.AsType here because the testdata Go version may not
// have it, but the analyzer only flags errors.As calls.
