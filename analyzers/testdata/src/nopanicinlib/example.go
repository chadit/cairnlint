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
