package analyzers

import "golang.org/x/tools/go/analysis"

// contextBackgroundAnalyzer returns an analyzer that flags
// context.Background() in test files. Tests should use t.Context()
// which is canceled when the test ends, preventing goroutine leaks.
//
// t.Context() arrived in Go 1.24, so the rule is gated on that version and
// stays quiet where no testing value can supply one or where it is already
// canceled; see testCallConfig.suggestsTContext. A Background() that is the
// base of a context.With* call is left to wrappedcontextbackground, which
// reports that line with the more specific message.
func contextBackgroundAnalyzer() *analysis.Analyzer {
	return testCallAnalyzer(&testCallConfig{
		name:             "contextbackground",
		doc:              "flags context.Background() in test files; use t.Context() instead",
		message:          "use t.Context() instead of context.Background() in tests",
		matchers:         []callMatcher{{pkgPath: contextPkgPath, funcName: "Background"}},
		skipWhenArgOf:    contextWrapperMatchers(),
		suggestsTContext: true,
	})
}
