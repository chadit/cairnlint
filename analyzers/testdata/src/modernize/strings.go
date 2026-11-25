package modernize

import "strings"

// Key returns the text before the first equals sign, using the Index and
// slice pair that strings.Cut replaces.
func Key(pair string) string {
	idx := strings.Index(pair, "=")
	if idx >= 0 {
		return pair[:idx]
	}

	return pair
}

// Position only reports where the sign is, so stringscut stays quiet.
func Position(pair string) int {
	return strings.Index(pair, "=")
}

// Unprefixed trims a prefix it first tested for, the pair CutPrefix replaces.
func Unprefixed(name string) (string, bool) {
	if strings.HasPrefix(name, "x-") {
		return strings.TrimPrefix(name, "x-"), true
	}

	return name, false
}

// Prefixed only tests for the prefix, so stringscutprefix stays quiet.
func Prefixed(name string) bool {
	return strings.HasPrefix(name, "x-")
}
