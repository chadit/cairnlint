package typeassertnocheck

import "io"

// Flagged: single-value type assertion panics on failure.
func uncheckedAssert(val any) {
	s := val.(string) // want `unchecked type assertion; use the comma-ok form`
	_ = s
}

// Flagged: unchecked assertion to interface type.
func uncheckedInterfaceAssert(val any) {
	r := val.(io.Reader) // want `unchecked type assertion; use the comma-ok form`
	_ = r
}

// Not flagged: comma-ok form is safe.
func checkedAssert(val any) {
	s, isString := val.(string)
	_ = s
	_ = isString
}

// Not flagged: type switch is safe.
func typeSwitchAssert(val any) {
	switch v := val.(type) {
	case string:
		_ = v
	case int:
		_ = v
	}
}
