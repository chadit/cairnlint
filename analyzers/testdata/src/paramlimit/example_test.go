package paramlimit

import (
	"net/http"
	"testing"
)

// Not reported: testing handles do not count.
func assertUser(t testing.TB, name, email string, age int) {
	t.Helper()
}

func assertUserT(t *testing.T, name, email string, age int) {
	t.Helper()
}

func benchUser(b *testing.B, name, email string, age int) {
	b.Helper()
}

// Reported: test helpers are in scope, and 4 domain parameters remain.
func assertProfile(t *testing.T, name, email string, age int, role string) { // want `assertProfile has 4 domain parameters`
	t.Helper()
}

// Not reported: a fuzz handle does not count.
func fuzzUser(f *testing.F, name, email string, age int) {
	f.Helper()
}

// Not reported: the rule's ACCEPTABLE testing-handle example.
func assertCreateRequest(t testing.TB, r *http.Request, accountID string, defaults ProxyDefaults) {
	t.Helper()
}
