package httpstatusliteral

import (
	"net/http"
	"os"
)

// fallbackStatus is marked by this package's own http.Error call.
var fallbackStatus = 502 // want `literal HTTP status 502; use http\.StatusBadGateway` fallbackStatus:"httpStatusSink"

func fail(w http.ResponseWriter) {
	http.Error(w, "upstream failed", fallbackStatus)
}

// code is a local that WriteHeader receives directly, so writes to it count.
func handle(w http.ResponseWriter, found bool) {
	code := 404 // want `literal HTTP status 404; use http\.StatusNotFound`
	if found {
		code = 200 // want `literal HTTP status 200; use http\.StatusOK`
	}

	retry := 503 // want `literal HTTP status 503; use http\.StatusServiceUnavailable`

	w.Header().Set("Retry-After", "1")
	w.WriteHeader(code)
	w.WriteHeader(retry)
	w.WriteHeader(404)
}

// A parameter passed to WriteHeader is a variable like any other. Its
// callers pass call arguments, which are not reported.
func respond(w http.ResponseWriter, code int) {
	if code == 204 { // want `literal HTTP status 204; use http\.StatusNoContent`
		return
	}

	w.WriteHeader(code)
}

func callRespond(w http.ResponseWriter) {
	respond(w, 404)
}

// statusRecorder implements http.ResponseWriter, so its own WriteHeader
// method is a sink too.
type statusRecorder struct {
	http.ResponseWriter
	fallback int // want fallback:"httpStatusSink"
	status   int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *statusRecorder) finish() {
	rec.WriteHeader(rec.fallback)
}

func newRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w, fallback: 500} // want `literal HTTP status 500; use http\.StatusInternalServerError`
}

// tableWriter has a WriteHeader(int) method but is no ResponseWriter, so
// what it receives is a column count, not a status.
type tableWriter struct {
	columns int
}

func (tw tableWriter) WriteHeader(columns int) {
	_ = columns
}

func renderTable() {
	table := tableWriter{columns: 200}
	table.WriteHeader(table.columns)
}

// Reply is an SMTP reply. Its Status never reaches net/http, so 421 stays,
// even though net/http names 421 StatusMisdirectedRequest.
type Reply struct {
	Status int
	Text   string
}

func busy() Reply {
	reply := Reply{Status: 421, Text: "try again later"}
	if reply.Status == 421 {
		reply.Text = "service not available"
	}

	return reply
}

// StatusTimeout is an exit code, which shares the Status name and nothing else.
var StatusTimeout = 124

func exitOnTimeout() {
	os.Exit(StatusTimeout)
}
