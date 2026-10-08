package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// wrappedContextBackgroundAnalyzer returns an analyzer that flags the context
// derivation functions wrapping context.Background() in test files. Even
// wrapped, the base context should be t.Context() so it is canceled at the
// end of the test. WithoutCancel is left out because its result is never
// canceled whatever the base; contextbackground still reports the inner
// call. The gate and scope rules match contextbackground, since the advice
// is the same.
func wrappedContextBackgroundAnalyzer() *analysis.Analyzer {
	return testCallAnalyzer(&testCallConfig{
		name:             "wrappedcontextbackground",
		doc:              "flags context.With*/AfterFunc(context.Background()) in test files; use t.Context() as base",
		message:          "use t.Context() as base context in tests, not context.Background()",
		matchers:         contextWrapperMatchers(),
		argMatches:       hasBackgroundArg,
		suggestsTContext: true,
	})
}

// contextWrapperMatchers lists the context derivation functions this analyzer
// reports. contextbackground reads the same list to leave those calls' inner
// context.Background() to this analyzer, so each line is reported once.
func contextWrapperMatchers() []callMatcher {
	return []callMatcher{
		{pkgPath: contextPkgPath, funcName: "WithCancel"},
		{pkgPath: contextPkgPath, funcName: "WithCancelCause"},
		{pkgPath: contextPkgPath, funcName: "WithTimeout"},
		{pkgPath: contextPkgPath, funcName: "WithTimeoutCause"},
		{pkgPath: contextPkgPath, funcName: "WithDeadline"},
		{pkgPath: contextPkgPath, funcName: "WithDeadlineCause"},
		{pkgPath: contextPkgPath, funcName: "AfterFunc"},
	}
}

// hasBackgroundArg reports whether the first argument to call is
// context.Background().
func hasBackgroundArg(call *ast.CallExpr, info *types.Info) bool {
	if len(call.Args) == 0 {
		return false
	}

	argCall, isCall := call.Args[0].(*ast.CallExpr)
	if !isCall {
		return false
	}

	return isCallTo(argCall, info, contextPkgPath, "Background")
}
