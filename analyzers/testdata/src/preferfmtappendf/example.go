package preferfmtappendf

import (
	"bytes"
	"fmt"
	"io"
)

// Not flagged: fmt.Appendf(nil, ...) reads less clearly than the conversion
// and returns nil instead of an empty slice on empty output.
func bareConversion(name string) []byte {
	return []byte(fmt.Sprintf("hello, %s", name))
}

// Not flagged: fmt.Appendf is the preferred approach.
func goodAppend(name string) []byte {
	return fmt.Appendf(nil, "hello, %s", name)
}

// Not flagged: []byte conversion of a non-Sprintf expression.
func plainConversion(msg string) []byte {
	return []byte(msg)
}

// Not flagged: fmt.Sprintf without []byte conversion.
func justSprintf(name string) string {
	return fmt.Sprintf("hello, %s", name)
}

// Flagged with the Fprintf wording: the byte slice exists only to satisfy
// Write, so Fprintf drops both the string and the slice.
func writeToWriter(dst io.Writer, name string) {
	_, _ = dst.Write([]byte(fmt.Sprintf("hello, %s", name))) // want `use fmt\.Fprintf\(dst, \.\.\.\) instead of dst\.Write`
}

// Flagged with the Fprintf wording: a concrete writer has the same shape.
func writeToBuffer(buf *bytes.Buffer, name string) {
	_, _ = buf.Write([]byte(fmt.Sprintf("hello, %s", name))) // want `use fmt\.Fprintf\(buf, \.\.\.\) instead of buf\.Write`
}

type recorder struct{}

func (recorder) Write(payload []byte) error { return nil }

// Not flagged: recorder.Write lacks the io.Writer shape, so Fprintf cannot
// replace it, and the bare conversion is not reported.
func writeToNonWriter(rec recorder, name string) {
	_ = rec.Write([]byte(fmt.Sprintf("hello, %s", name)))
}

// Flagged with the Appendf wording: Appendf writes straight into dst.
func appendToVariable(dst []byte, n int) []byte {
	return append(dst, []byte(fmt.Sprintf("x %d", n))...) // want `use fmt\.Appendf\(dst, \.\.\.\) instead of append\(dst, \[\]byte\(fmt\.Sprintf\(\.\.\.\)\)\.\.\.\)`
}

func buf() []byte { return nil }

// Flagged: the message names the destination expression as written.
func appendToCall(n int) []byte {
	return append(buf(), []byte(fmt.Sprintf("x %d", n))...) // want `use fmt\.Appendf\(buf\(\), \.\.\.\) instead of append\(buf\(\), `
}

// Not flagged: without the spread the conversion is a single element, so
// this only compiles for a [][]byte destination, which Appendf cannot fill.
func appendWithoutSpread(lines [][]byte, n int) [][]byte {
	return append(lines, []byte(fmt.Sprintf("x %d", n)))
}

// Not flagged: a local function named append is not the builtin.
func shadowedAppend(dst []byte, n int) []byte {
	append := func(head []byte, tail ...byte) []byte { return head }

	return append(dst, []byte(fmt.Sprintf("x %d", n))...)
}
