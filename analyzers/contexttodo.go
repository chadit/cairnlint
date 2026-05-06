package analyzers

import "golang.org/x/tools/go/analysis"

// contextTODOAnalyzer returns an analyzer that flags the context
// placeholder constructor in test files. Tests should use t.Context()
// which is canceled when the test ends.
func contextTODOAnalyzer() *analysis.Analyzer {
	return testCallWithSynctestExemption(synctestExemptConfig{
		name:     "contexttodo",
		doc:      "flags context.TODO() in test files; use t.Context() instead",
		message:  "use t.Context() instead of context.TODO() in tests",
		matchers: []callMatcher{{pkgPath: contextPkgPath, funcName: "TODO"}},
	})
}
