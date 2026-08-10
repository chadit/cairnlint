package redundantbodydrain

import (
	"io"
	"net/http"
)

// Flagged: the copy result is discarded, so the drain exists only for reuse.
func drainAndClose(resp *http.Response) {
	io.Copy(io.Discard, resp.Body) // want `Response\.Body drains unread content on Close from Go 1\.27`
	resp.Body.Close()
}

// Flagged: assigning to blanks discards the results just the same.
func drainToBlanks(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body) // want `Response\.Body drains unread content on Close from Go 1\.27`
	resp.Body.Close()
}

// Not flagged: the caller keeps the byte count, so the copy does more than
// prepare the connection for reuse.
func measureRemaining(resp *http.Response) int64 {
	size, _ := io.Copy(io.Discard, resp.Body)

	return size
}

// Not flagged: the body goes somewhere real.
func saveBody(resp *http.Response, dst io.Writer) error {
	_, err := io.Copy(dst, resp.Body)

	return err
}

// Not flagged: not an http.Response body.
func drainReader(src io.Reader) {
	io.Copy(io.Discard, src)
}
