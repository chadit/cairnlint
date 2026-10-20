package analyzers

import "golang.org/x/tools/go/analysis"

// contextBackgroundAnalyzer flags context.Background() in test files, where
// t.Context() is canceled when the test ends and so prevents goroutine leaks.
// It needs Go 1.24 and stays quiet where t.Context() is unavailable or already
// canceled (see testCallConfig.suggestsTContext). A Background() that bases a
// context.With* call is left to wrappedcontextbackground's sharper message.
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
