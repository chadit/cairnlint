// Package gomod121 is loaded in module mode with go 1.21 in its go.mod, so
// rules gated on a newer Go version must stay quiet here.
package gomod121

import (
	"errors"
	"io/fs"
)

// IsPathError reports whether err wraps a path error. errors.AsType needs Go
// 1.26, so prefererrorsastype must not suggest it.
func IsPathError(err error) bool {
	var pathErr *fs.PathError

	return errors.As(err, &pathErr)
}

// Sum adds the integers below limit. A range over an integer needs Go 1.22,
// so rangeint must not suggest it.
func Sum(limit int) int {
	var total int

	for i := 0; i < limit; i++ {
		total += i
	}

	return total
}

// Version names the module's Go version — the em dash is the finding that
// shows this run reports at all.
const Version = "go1.21"
