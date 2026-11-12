// Package cloudtest stands in for a fake cloud server whose test hooks pick
// the status it answers with. It declares the fields and variables, and it
// is the package that hands them to net/http, so it exports their marks.
package cloudtest

import "net/http"

// Directive tells the fake server how to answer one request.
type Directive struct {
	Path           string
	StatusOverride int // want StatusOverride:"httpStatusSink"
}

// Server replays one directive.
type Server struct {
	Directive Directive
}

func (s *Server) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(s.Directive.StatusOverride)
}

// Default is set in the declaring package itself, which reads its own marks.
func Default() Directive {
	return Directive{StatusOverride: 200} // want `literal HTTP status 200; use http\.StatusOK`
}

// FailureStatus reaches http.Error.
var FailureStatus = http.StatusServiceUnavailable // want FailureStatus:"httpStatusSink"

// RedirectStatus reaches http.Redirect.
var RedirectStatus int // want RedirectStatus:"httpStatusSink"

func Fail(w http.ResponseWriter) {
	http.Error(w, "unavailable", FailureStatus)
}

func Move(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/elsewhere", RedirectStatus)
}

// Canned reaches http.Response.StatusCode through an assignment.
type Canned struct {
	Code int // want Code:"httpStatusSink"
	Body string
}

func (c Canned) Response() *http.Response {
	resp := &http.Response{}
	resp.StatusCode = c.Code

	return resp
}

// Recorded reaches http.Response.StatusCode through a composite literal.
type Recorded struct {
	Status int // want Status:"httpStatusSink"
}

func (r Recorded) Response() *http.Response {
	return &http.Response{StatusCode: r.Status}
}

// Indirect copies its field into a local before writing it. The local is
// what WriteHeader receives, so the field carries no mark.
type Indirect struct {
	StatusOverride int
}

func (d Indirect) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	code := d.StatusOverride
	w.WriteHeader(code)
}

// teapot is a constant, so it is never marked and its declaration is left alone.
const teapot = 418

func Brew(w http.ResponseWriter) {
	w.WriteHeader(teapot)
}
