package httpclienttimeout

import (
	"net/http"
	"time"
)

// BadNoTimeout creates client without Timeout.
func BadNoTimeout() *http.Client {
	return &http.Client{} // want `http\.Client without Timeout`
}

// BadWithTransportOnly has other fields but no Timeout.
func BadWithTransportOnly() *http.Client {
	return &http.Client{ // want `http\.Client without Timeout`
		Transport: http.DefaultTransport,
	}
}

// GoodWithTimeout has Timeout set.
func GoodWithTimeout() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}

// GoodWithTimeoutAndTransport has both.
func GoodWithTimeoutAndTransport() *http.Client {
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: http.DefaultTransport,
	}
}
