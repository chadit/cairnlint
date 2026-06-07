package testhelper

import "testing"

// Flagged: takes *testing.T but does not mark itself as a helper.
func helperBad(t *testing.T) { // want `helper "helperBad" does not call`
	t.Log("no helper call")
}

// Not flagged: calls t.Helper() first.
func helperGood(t *testing.T) {
	t.Helper()
	t.Log("ok")
}

// Not flagged: a test entry point is not a helper.
func TestSomething(t *testing.T) {
	helperGood(t)
}

// Not flagged: no testing parameter.
func notAHelper() {}

// Not flagged: blank testing parameter cannot call Helper.
func helperBlank(_ *testing.T) {}
