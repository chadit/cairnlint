package prefervarzero

// Flagged: short declaration with zero-value string.
func badString() string {
	s := "" // want `use "var s string" instead of short declaration with zero value`
	return s
}

// Flagged: short declaration with zero-value int.
func badInt() int {
	n := 0 // want `use "var n int" instead of short declaration with zero value`
	return n
}

// Flagged: short declaration with zero-value bool.
func badBool() bool {
	b := false // want `use "var b bool" instead of short declaration with zero value`
	return b
}

// Not flagged: var declarations with zero value (idiomatic).
func goodDeclarations() (string, int, bool) {
	var s string
	var n int
	var b bool

	return s, n, b
}

// Not flagged: short declaration with non-zero values.
func nonZeroValues() (string, int, bool) {
	s := "hello"
	n := 42
	b := true

	return s, n, b
}

// Not flagged: multi-assignment.
func multiAssign() (int, int) {
	x, y := 0, 1

	return x, y
}
