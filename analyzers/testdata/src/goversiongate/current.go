//go:build go1.26

package goversiongate

import (
	"errors"
	"os"
)

// Flagged: this file's language version has errors.AsType, so the suggestion
// is safe to act on even though a sibling file in the same package is pinned
// lower.
func currentUnwrap(err error) bool {
	var pathErr *os.PathError

	return errors.As(err, &pathErr) // want `use errors\.AsType\[T\]\(err\) instead of errors\.As`
}
