package agentstubbody

import (
	"errors"
	"fmt"
)

// Store is a fixture receiver type.
type Store struct{}

// Box carries a value so its accessor returns real data.
type Box struct{ n int }

// noopCloser is an intentional no-op, the kind that must NOT be flagged.
type noopCloser struct{}

// ValidateConfig reports the first invalid field it finds.
func ValidateConfig(c int) error { // want `returns without doing the work its signature implies`
	return nil
}

// LoadUser fetches the user identified by id.
func LoadUser(id string) (*int, error) { // want `stub: wire up a real implementation`
	return nil, errors.New("not implemented")
}

// Flush writes buffered data to the underlying store.
func Flush() { // want `has an empty body but its doc describes behavior`
}

// Size returns the configured buffer size in bytes.
func Size() int { // want `returns a canned value; verify it does what its doc claims`
	return 0
}

// Save persists the record.
func (s *Store) Save() error { // want `stub: wire up a real implementation`
	return fmt.Errorf("TODO: implement Save")
}

// Add returns the sum of a and b.
func Add(a, b int) int {
	sum := a + b
	return sum
}

// Double returns twice x.
func Double(x int) int {
	return x * 2
}

// N returns the stored value.
func (b Box) N() int {
	return b.n
}

// CheckPositive errors when x is negative.
func CheckPositive(x int) error {
	if x < 0 {
		return errors.New("negative")
	}

	return nil
}

// NewStore builds a Store.
func NewStore() *Store {
	return &Store{}
}

func (noopCloser) Close() error { return nil } // intentional no-op: no params, no method doc, so not flagged

// Setup prepares the environment and reports the first failure it hits.
func Setup() (err error) { // want `returns without doing the work its signature implies`
	return
}

// Coords returns the configured x and y position.
func Coords() (int, int) { // want `returns a canned value; verify it does what its doc claims`
	return 0, 0
}

// TestWidget exercises the widget end to end. A Test-framework name, so even
// with an empty body and a doc it must NOT be flagged.
func TestWidget() {}

func noop() {
}
