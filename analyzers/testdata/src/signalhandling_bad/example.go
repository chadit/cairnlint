package main

import (
	"net/http"
)

func main() { // want `main\(\) starts a server without signal handling; use signal\.NotifyContext for graceful shutdown`
	http.ListenAndServe(":8080", nil) // #nosec G104 G114 -- test fixture
}
