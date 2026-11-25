package modernize

import (
	"slices"
	"sort"
	"strings"
)

// Contains reports whether target is in values. The loop is what
// slicescontains rewrites to slices.Contains.
func Contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}

// CountMatches counts every match, so no single slices call replaces the
// loop and slicescontains stays quiet.
func CountMatches(values []string, target string) int {
	var count int

	for _, value := range values {
		if value == target {
			count++
		}
	}

	return count
}

// SortInts sorts with a less function that slices.Sort already implies.
func SortInts(values []int) {
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
}

// SortDescending sorts in reverse, which slices.Sort cannot express, so
// slicessort stays quiet.
func SortDescending(values []int) {
	sort.Slice(values, func(i, j int) bool { return values[i] > values[j] })
}

// Clip drops spare capacity with a three-index slice.
func Clip(values []byte) []byte {
	return values[:len(values):len(values)]
}

// ClipOther caps at another slice's length, which slices.Clip cannot express.
func ClipOther(values, other []byte) []byte {
	return values[:len(other):len(other)]
}

// Reverse visits values back to front with an index loop.
func Reverse(values []int) []int {
	out := make([]int, 0, len(values))

	for i := len(values) - 1; i >= 0; i-- {
		out = append(out, values[i])
	}

	return out
}

// EveryOther steps by two, which slices.Backward cannot express.
func EveryOther(values []int) []int {
	out := make([]int, 0, len(values))

	for i := len(values) - 1; i >= 0; i -= 2 {
		out = append(out, values[i])
	}

	return out
}

// Fields ranges over the slice strings.Fields builds only to discard it.
func Fields(line string) int {
	var count int

	for _, field := range strings.Fields(line) {
		count += len(field)
	}

	return count
}

// FieldsKept keeps the slice, so a sequence would not do and stringsseq
// stays quiet.
func FieldsKept(line string) []string {
	fields := strings.Fields(line)
	slices.Sort(fields)

	return fields
}
