package modernize

import "strings"

// Sum counts up with a three-clause loop that rangeint rewrites.
func Sum(limit int) int {
	var total int

	for i := 0; i < limit; i++ {
		total += i
	}

	return total
}

// SumEvens steps by two, which a range over an integer cannot express.
func SumEvens(limit int) int {
	var total int

	for i := 0; i < limit; i += 2 {
		total += i
	}

	return total
}

// Upper re-declares the loop variable, which Go 1.22 made redundant.
func Upper(words []string) []func() string {
	out := make([]func() string, 0, len(words))

	for _, word := range words {
		word := word
		out = append(out, func() string { return strings.ToUpper(word) })
	}

	return out
}

// Trimmed declares a new variable from the loop variable, which is not a
// re-declaration, so forvar stays quiet.
func Trimmed(words []string) []string {
	out := make([]string, 0, len(words))

	for _, word := range words {
		trimmed := strings.TrimSpace(word)
		out = append(out, trimmed)
	}

	return out
}

// Larger picks the bigger value with an if statement that min and max replace.
func Larger(left, right int) int {
	bigger := left
	if right > left {
		bigger = right
	}

	return bigger
}

// LargerLogged does extra work in the branch, so minmax stays quiet.
func LargerLogged(left, right int, log *[]string) int {
	bigger := left
	if right > left {
		bigger = right
		*log = append(*log, "right")
	}

	return bigger
}

// Merge copies every entry with a loop that maps.Copy replaces.
func Merge(dst, src map[string]int) {
	for key, value := range src {
		dst[key] = value
	}
}

// MergeDoubled transforms each value, so no maps call replaces the loop.
func MergeDoubled(dst, src map[string]int) {
	for key, value := range src {
		dst[key] = value * 2
	}
}
