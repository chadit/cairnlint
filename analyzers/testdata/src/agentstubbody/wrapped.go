package agentstubbody

import (
	"errors"
	"fmt"
)

var errClosed = errors.New("closed")

// OpenConfig reports why the config file could not be opened.
func OpenConfig(err error) error {
	return fmt.Errorf("open config: %w", err)
}

// CloseStore reports the store as closed.
func CloseStore(name string) error {
	return fmt.Errorf("%w: %s", errClosed, name)
}

// Reserve claims the named slot.
func Reserve(name string) error { // want `stub: wire up a real implementation`
	return fmt.Errorf("not implemented")
}

// Limit reports the configured limit as exceeded.
func Limit() error { // want `returns without doing the work its signature implies`
	return fmt.Errorf("limit %d exceeded", 3)
}

// Prepare readies the named job.
func Prepare(name string) error { // want `stub: wire up a real implementation`
	return fmt.Errorf("not implemented: %s", name)
}

// Dial connects to host.
func Dial(host string) error { // want `stub: wire up a real implementation`
	return fmt.Errorf("%s: unimplemented", host)
}

// Shutdown closes the store.
func Shutdown() error { // want `returns without doing the work its signature implies`
	return fmt.Errorf("%w: store", errClosed)
}

// Inspect reports the inspected value.
func Inspect() error { // want `returns without doing the work its signature implies`
	return fmt.Errorf("nil %v", nil)
}
