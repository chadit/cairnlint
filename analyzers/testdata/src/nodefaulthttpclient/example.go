package nodefaulthttpclient

import "net/http"

// Flagged: direct reference to http.DefaultClient.
func UseDefaultClient(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req) // #nosec G704 -- test fixture // want `http\.DefaultClient has no timeout; construct an http\.Client with explicit Timeout`
}

// Flagged: assigning DefaultClient to a variable.
func AssignDefaultClient() *http.Client {
	client := http.DefaultClient // want `http\.DefaultClient has no timeout; construct an http\.Client with explicit Timeout`
	return client
}

// Not flagged: custom client with timeout.
func UseCustomClient(req *http.Request) (*http.Response, error) {
	client := &http.Client{}
	return client.Do(req) // #nosec G704 -- test fixture
}

// Not flagged: a struct field named DefaultClient is not the http package variable.
type wrapper struct {
	DefaultClient string
}

func fieldAccess() string {
	w := wrapper{DefaultClient: "ok"}
	return w.DefaultClient
}
