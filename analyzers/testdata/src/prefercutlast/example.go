package prefercutlast

import (
	"bytes"
	"strings"
)

// Flagged: the LastIndex result becomes both slice bounds, which is what
// CutLast returns directly.
func splitExtension(path string) (string, string) {
	idx := strings.LastIndex(path, ".") // want `strings\.LastIndex followed by slicing can be replaced with strings\.CutLast`
	if idx < 0 {
		return path, ""
	}

	return path[:idx], path[idx+1:]
}

// Flagged: bytes carries the same pair.
func splitTrailer(payload, sep []byte) ([]byte, []byte) {
	idx := bytes.LastIndex(payload, sep) // want `bytes\.LastIndex followed by slicing can be replaced with bytes\.CutLast`
	if idx < 0 {
		return payload, nil
	}

	return payload[:idx], payload[idx+len(sep):]
}

// Flagged: only the tail half is taken, which CutLast still covers.
func tailOnly(path string) string {
	idx := strings.LastIndex(path, "/") // want `strings\.LastIndex followed by slicing can be replaced with strings\.CutLast`
	if idx < 0 {
		return path
	}

	return path[idx+1:]
}

// Not flagged: the index is compared, never used as a slice bound.
func endsWithSeparator(path string) bool {
	idx := strings.LastIndex(path, "/")

	return idx == len(path)-1
}

// Not flagged: Index rather than LastIndex. strings.Cut already covers it.
func splitFirst(path string) string {
	idx := strings.Index(path, "/")
	if idx < 0 {
		return path
	}

	return path[:idx]
}

// Not flagged: the slice operand is a different string from the one indexed,
// so CutLast on path would not produce this result.
func sliceOtherOperand(path, other string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 || idx > len(other) {
		return other
	}

	return other[:idx]
}

// Not flagged: the result is sliced with a bound that never mentions the index.
func unrelatedSlice(path string) string {
	idx := strings.LastIndex(path, "/")
	_ = idx

	return path[1:]
}
