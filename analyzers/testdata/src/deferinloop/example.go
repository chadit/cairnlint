package deferinloop

import "os"

// BadDeferInForLoop shows a classic resource leak pattern.
func BadDeferInForLoop(paths []string) {
	for _, path := range paths {
		f, err := os.Open(path) // #nosec G304 -- test fixture
		if err != nil {
			continue
		}
		defer f.Close() // want `defer inside loop; deferred calls execute when the function returns, not per iteration`
	}
}

// BadDeferInCStyleLoop uses a traditional for loop.
func BadDeferInCStyleLoop(n int) {
	for idx := 0; idx < n; idx++ {
		f, err := os.CreateTemp("", "test")
		if err != nil {
			continue
		}
		defer f.Close() // want `defer inside loop; deferred calls execute when the function returns, not per iteration`
	}
}

// GoodDeferOutsideLoop is fine because the deferred call is at function scope.
func GoodDeferOutsideLoop() {
	f, err := os.Open("file.txt") // #nosec G304 -- test fixture
	if err != nil {
		return
	}
	defer f.Close()
}

// GoodClosureInsideLoop wraps the body in a closure so each deferred call
// belongs to the closure's scope, not the outer function.
func GoodClosureInsideLoop(paths []string) {
	for _, path := range paths {
		func() {
			f, err := os.Open(path) // #nosec G304 -- test fixture
			if err != nil {
				return
			}
			defer f.Close()
		}()
	}
}
