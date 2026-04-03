package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// wrappedContextBackgroundAnalyzer returns an analyzer that flags
// context.WithCancel/WithTimeout/WithDeadline wrapping context.Background()
// in test files. Even wrapped, the base context should be t.Context()
// so it is canceled when the test ends.
func wrappedContextBackgroundAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "wrappedcontextbackground",
		Doc:      "flags context.With*(context.Background()) in test files; use t.Context() as base",
		Run:      runWrappedContextBackground,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runWrappedContextBackground(pass *analysis.Pass) (any, error) {
	wrapperFuncs := []callMatcher{
		{pkgPath: "context", funcName: "WithCancel"},
		{pkgPath: "context", funcName: "WithTimeout"},
		{pkgPath: "context", funcName: "WithDeadline"},
	}

	if !hasTestFiles(pass) {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.WithStack(nodeFilter, func(node ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}

		if !isTestFile(pass, call) {
			return true
		}

		if !matchesAny(call, pass.TypesInfo, wrapperFuncs) {
			return true
		}

		if !hasBackgroundArg(call, pass) {
			return true
		}

		if isInsideSynctestClosure(stack, pass.TypesInfo) {
			return true
		}

		pass.Reportf(call.Pos(), "use t.Context() as base context in tests, not context.Background()")

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// hasBackgroundArg reports whether the first argument to call is
// context.Background().
func hasBackgroundArg(call *ast.CallExpr, pass *analysis.Pass) bool {
	if len(call.Args) == 0 {
		return false
	}

	argCall, isCall := call.Args[0].(*ast.CallExpr)
	if !isCall {
		return false
	}

	return isCallTo(argCall, pass.TypesInfo, "context", "Background")
}
