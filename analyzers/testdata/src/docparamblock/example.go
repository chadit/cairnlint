package docparamblock

// BadFunc has a Javadoc-style parameter block.
//
// Parameters: // want `Javadoc-style section header`
//   - name: the caller name
//
// Returns: // want `Javadoc-style section header`
//   - string: the greeting
func BadFunc(name string) string {
	return "hi " + name
}

// GoodFunc weaves its parameters into prose, which is the Go convention.
// The name argument is used in the returned greeting.
func GoodFunc(name string) string {
	return "hi " + name
}

// placeholder keeps the package compilable.
func placeholder() {}
