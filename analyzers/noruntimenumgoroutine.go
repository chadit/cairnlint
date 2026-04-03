package analyzers

import "golang.org/x/tools/go/analysis"

// noRuntimeNumGoroutineAnalyzer returns an analyzer that flags
// runtime.NumGoroutine() calls in test files. This function is
// unreliable for leak detection because the runtime goroutine count
// includes unrelated goroutines; use goleak instead.
func noRuntimeNumGoroutineAnalyzer() *analysis.Analyzer {
	return testCallWithSynctestExemption(synctestExemptConfig{
		name:     "noruntimenumgoroutine",
		doc:      "flags runtime.NumGoroutine() in test files; use goleak for goroutine leak detection",
		message:  "runtime.NumGoroutine() is unreliable for leak detection; use goleak instead",
		matchers: []callMatcher{{pkgPath: "runtime", funcName: "NumGoroutine"}},
	})
}
