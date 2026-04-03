package discardedcontext

import "context"

// Bad: discarded context breaks cancellation chain.
func Handle(_ context.Context, data string) string { // want `discarded context\.Context breaks cancellation`
	return data
}

// Good: context is used.
func Process(ctx context.Context, data string) string {
	_ = ctx
	return data
}

// Bad: discarded in a function literal.
var fn = func(_ context.Context) {} // want `discarded context\.Context breaks cancellation`

// Good: no context parameter at all.
func NoContext(data string) string {
	return data
}
