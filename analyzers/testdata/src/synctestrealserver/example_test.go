package synctestrealserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
)

func handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}

// Flagged: a real listener sits outside the bubble.
func TestRealServerInBubble(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		srv := httptest.NewServer(handler()) // want `httptest\.NewServer inside synctest\.Test listens on a real socket`
		defer srv.Close()
	})
}

// Flagged: the TLS constructor has the same problem.
func TestRealTLSServerInBubble(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		srv := httptest.NewTLSServer(handler()) // want `httptest\.NewTLSServer inside synctest\.Test listens on a real socket`
		defer srv.Close()
	})
}

// Flagged: NewUnstartedServer already binds a listener.
func TestUnstartedServerInBubble(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		srv := httptest.NewUnstartedServer(handler()) // want `httptest\.NewUnstartedServer inside synctest\.Test listens on a real socket`
		srv.Start()
		defer srv.Close()
	})
}

// Not flagged: outside a bubble a real server is the right tool.
func TestRealServerOutsideBubble(t *testing.T) {
	srv := httptest.NewServer(handler())
	defer srv.Close()
}

// Not flagged: a recorder never touches the network.
func TestRecorderInBubble(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rec := httptest.NewRecorder()
		_ = rec
	})
}
