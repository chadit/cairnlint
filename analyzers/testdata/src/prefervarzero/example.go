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

// Not flagged: an init clause cannot hold a var declaration, so the short
// form is the only way to write these.
func initClauses(limit int, value any) int {
	var total int

	for i := 0; i < limit; i++ {
		total += i
	}

	if n := 0; n < limit {
		total += n
	}

	switch s := ""; s {
	case "":
		total++
	}

	switch b := false; typed := value.(type) {
	case int:
		_ = b
		total += typed
	}

	return total
}

// Flagged: a short declaration inside a loop body can still be a var.
func insideLoop(limit int) int {
	var total int

	for range limit {
		n := 0 // want `use "var n int" instead of short declaration with zero value`
		total += n
	}

	return total
}
