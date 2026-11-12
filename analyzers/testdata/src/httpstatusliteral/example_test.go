package httpstatusliteral

import (
	"httpstatusliteral/cloudtest"
	"net/http"
	"testing"
)

// The StatusOverride: 404 line is the shape that motivated the analyzer: a
// test in one package sets a field that another package writes as a status.
func TestMissingObject(t *testing.T) {
	directive := cloudtest.Directive{Path: "/object", StatusOverride: 404} // want `literal HTTP status 404; use http\.StatusNotFound`

	server := &cloudtest.Server{Directive: directive}
	if server.Directive.StatusOverride == 404 { // want `literal HTTP status 404; use http\.StatusNotFound`
		t.Log("missing")
	}
}

func TestPositionalAndElided(t *testing.T) {
	positional := cloudtest.Directive{"/object", 503}       // want `literal HTTP status 503; use http\.StatusServiceUnavailable`
	elided := []*cloudtest.Directive{{StatusOverride: 429}} // want `literal HTTP status 429; use http\.StatusTooManyRequests`

	t.Log(positional, elided)
}

func TestAssignAndCompare(t *testing.T) {
	var directive cloudtest.Directive

	directive.StatusOverride = 500 // want `literal HTTP status 500; use http\.StatusInternalServerError`
	directive.StatusOverride = 410 // want `literal HTTP status 410; use http\.StatusGone`

	if 404 != directive.StatusOverride { // want `literal HTTP status 404; use http\.StatusNotFound`
		t.Log("not missing")
	}

	switch directive.StatusOverride {
	case 404, 410: // want `literal HTTP status 404; use http\.StatusNotFound` `literal HTTP status 410; use http\.StatusGone`
		t.Log("gone")
	case 299:
		t.Log("unnamed")
	}
}

func TestPackageVariables(t *testing.T) {
	cloudtest.FailureStatus = 502  // want `literal HTTP status 502; use http\.StatusBadGateway`
	cloudtest.RedirectStatus = 302 // want `literal HTTP status 302; use http\.StatusFound`

	if cloudtest.RedirectStatus == 307 { // want `literal HTTP status 307; use http\.StatusTemporaryRedirect`
		t.Log("temporary")
	}
}

func TestResponseStatusCodeFields(t *testing.T) {
	canned := cloudtest.Canned{Code: 201, Body: "{}"} // want `literal HTTP status 201; use http\.StatusCreated`
	recorded := cloudtest.Recorded{Status: 204}       // want `literal HTTP status 204; use http\.StatusNoContent`

	t.Log(canned, recorded)
}

// Each case below reaches a marked field without a value net/http names, or
// writes a field nothing marks.
func TestNotReported(t *testing.T) {
	const overrideStatus = 404

	cases := []cloudtest.Directive{
		{StatusOverride: 299},
		{StatusOverride: 306},
		{StatusOverride: 600},
		{StatusOverride: 99},
		{StatusOverride: http.StatusNotFound},
		{StatusOverride: overrideStatus},
	}

	indirect := cloudtest.Indirect{StatusOverride: 404}

	var directive cloudtest.Directive
	if directive.StatusOverride >= 500 {
		t.Log("server error")
	}

	t.Log(cases, indirect)
}
