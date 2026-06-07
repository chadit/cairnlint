package prefercolonequals

func F() {
	var x = 42 // want `use x :=`
	_ = x

	var s = "hello" // want `use s :=`
	_ = s

	// Not flagged: zero value is left for the var-with-zero convention.
	var z = 0
	_ = z

	// Not flagged: already the short form.
	y := 99
	_ = y

	// Not flagged: zero-value declaration without an initializer.
	var t int
	_ = t

	// Not flagged: multiple names.
	var a, b = 1, 2
	_, _ = a, b
}
