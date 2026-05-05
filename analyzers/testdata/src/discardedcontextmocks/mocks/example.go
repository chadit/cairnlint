package mocks

import "context"

// Mock implementations frequently need to satisfy interface signatures
// without using the context. The analyzer should not fire here because
// the file lives inside a `mocks/` directory.
type MockHandler struct{}

func (MockHandler) Handle(_ context.Context, data string) string {
	return data
}

var Fn = func(_ context.Context) {}
