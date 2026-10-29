package nopanicinlib

// Flagged: panic in library code.
func MustParse(input string) int {
	if input == "" {
		panic("empty input") // want `panic\(\) in library code; return an error instead`
	}

	return len(input)
}

// Flagged: panic with no argument context.
func BadInit() {
	panic("initialization failed") // want `panic\(\) in library code; return an error instead`
}

// Not flagged: no panic call.
func GoodParse(input string) (int, error) {
	if input == "" {
		return 0, nil
	}

	return len(input), nil
}

// Not flagged: init may fail fast, since a broken package cannot be used.
func init() {
	if len(registry) > 0 {
		panic("registry populated before init")
	}
}

var registry = map[string]func(){}

// Flagged: the closure runs whenever a caller invokes it, not during init.
func init() {
	registry["worker"] = func() {
		panic("worker failed") // want `panic\(\) in library code; return an error instead`
	}
}

// Flagged: a function named main outside package main is ordinary code.
func main() {
	panic("not an entry point") // want `panic\(\) in library code; return an error instead`
}

type loader struct{}

// Flagged: a method named init is not the package initializer.
func (loader) init() {
	panic("method, not initializer") // want `panic\(\) in library code; return an error instead`
}
