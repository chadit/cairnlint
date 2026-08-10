package preferfmtappendf

import (
	"bytes"
	"fmt"
	"io"
)

// Flagged: []byte(fmt.Sprintf(...)) has an unnecessary intermediate string.
func badConversion(name string) []byte {
	return []byte(fmt.Sprintf("hello, %s", name)) // want `use fmt\.Appendf\(nil, \.\.\.\) instead of \[\]byte\(fmt\.Sprintf`
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

// recorder has a Write method that is not io.Writer's, so the Appendf wording
// applies rather than the Fprintf one.
type recorder struct{}

func (recorder) Write(payload []byte) error { return nil }

func writeToNonWriter(rec recorder, name string) {
	_ = rec.Write([]byte(fmt.Sprintf("hello, %s", name))) // want `use fmt\.Appendf\(nil, \.\.\.\) instead of \[\]byte\(fmt\.Sprintf`
}
