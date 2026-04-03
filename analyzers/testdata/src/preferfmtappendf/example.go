package preferfmtappendf

import "fmt"

// Flagged: []byte(fmt.Sprintf(...)) has an unnecessary intermediate string.
func badConversion(name string) []byte {
	return []byte(fmt.Sprintf("hello, %s", name)) // want `use fmt\.Appendf\(nil, \.\.\.\) instead of \[\]byte\(fmt\.Sprintf`
}

// Not flagged: fmt.Appendf is the preferred approach.
func goodAppend(name string) []byte {
	return fmt.Appendf(nil, "hello, %s", name)
}

// Not flagged: []byte conversion of a non-Sprintf expression.
func plainConversion(msg string) []byte {
	return []byte(msg)
}

// Not flagged: fmt.Sprintf without []byte conversion.
func justSprintf(name string) string {
	return fmt.Sprintf("hello, %s", name)
}
