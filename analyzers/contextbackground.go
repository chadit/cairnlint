package analyzers

import "golang.org/x/tools/go/analysis"

// contextBackgroundAnalyzer returns an analyzer that flags
// context.Background() in test files. Tests should use t.Context()
// which is canceled when the test ends, preventing goroutine leaks.
func contextBackgroundAnalyzer() *analysis.Analyzer {
	return testCallWithSynctestExemption(synctestExemptConfig{
		name:     "contextbackground",
		doc:      "flags context.Background() in test files; use t.Context() instead",
		message:  "use t.Context() instead of context.Background() in tests",
		matchers: []callMatcher{{pkgPath: "context", funcName: "Background"}},
	})
}
