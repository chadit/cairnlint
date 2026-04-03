package sentinelerrors

import "errors"

// Bad: sentinel error outside errors.go.
var ErrNotFound = errors.New("not found") // want `sentinel error ErrNotFound should be declared in errors\.go`

// Bad: unexported sentinel error outside errors.go.
var errTimeout = errors.New("timeout") // want `sentinel error errTimeout should be declared in errors\.go`

// Good: not a sentinel error pattern (no Err/err prefix with uppercase).
var regularVar = errors.New("something")

// Good: not an errors.New call.
var ErrWrapped = fmt("wrapped")

func fmt(_ string) error { return nil }
