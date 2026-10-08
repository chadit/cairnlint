package analyzers

import "golang.org/x/tools/go/analysis"

// contextTODOAnalyzer returns an analyzer that flags the context
// placeholder constructor in test files. Tests should use t.Context()
// which is canceled when the test ends.
//
// t.Context() arrived in Go 1.24, so the rule is gated on that version and
// stays quiet where no testing value can supply one or where it is already
// canceled; see testCallConfig.suggestsTContext.
func contextTODOAnalyzer() *analysis.Analyzer {
	return testCallAnalyzer(&testCallConfig{
		name:             "contexttodo",
		doc:              "flags context.TODO() in test files; use t.Context() instead",
		message:          "use t.Context() instead of context.TODO() in tests",
		matchers:         []callMatcher{{pkgPath: contextPkgPath, funcName: "TODO"}},
		suggestsTContext: true,
	})
}
