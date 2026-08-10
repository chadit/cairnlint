//go:build go1.21

// Package goversiongate exercises the language-version gate. Both files sit in
// one package so a single pass sees two different file versions, which is what
// a //go:build constraint produces in a real module.
package goversiongate

import (
	"errors"
	"os"
)

// Not flagged: errors.AsType arrived in Go 1.26, and this file is pinned to
// go1.21. Suggesting it here would produce code that fails the stdversion vet
// check that `go test` runs by default from Go 1.27 on.
func legacyUnwrap(err error) bool {
	var pathErr *os.PathError

	return errors.As(err, &pathErr)
}
