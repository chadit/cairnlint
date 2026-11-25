package prefercolonequals

// Zero keeps a zero value in var form, which the var-with-zero convention asks
// for, so prefercolonequals leaves it alone.
func Zero() int {
	var z = 0

	return z
}
