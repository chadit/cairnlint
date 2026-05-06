package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// noErrStrContainsAnalyzer returns an analyzer that flags string-matching on
// error messages (e.g., strings.Contains(err.Error(), ...)) and testify
// equivalents. Matching on error strings is brittle; use errors.Is or
// errors.As for type-safe error checking.
func noErrStrContainsAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "noerrstrcontains",
		Doc:      "flags strings.Contains(err.Error(), ...) and testify equivalents; use errors.Is/errors.As",
		Run:      runNoErrStrContains,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runNoErrStrContains(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return
		}

		if hasErrDotErrorArg(call, pass) {
			pass.Reportf(call.Pos(), "do not match error strings with Contains(err.Error(), ...); use errors.Is or errors.As instead")
		}
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// hasErrDotErrorArg checks whether call is one of the flagged patterns:
//   - strings.Contains(err.Error(), ...)
//   - assert.Contains(t, err.Error(), ...)
//   - require.Contains(t, err.Error(), ...)
func hasErrDotErrorArg(call *ast.CallExpr, pass *analysis.Pass) bool {
	containsMatchers := []callMatcher{
		{pkgPath: stringsPkgPath, funcName: containsFunc},
		{pkgPath: "github.com/stretchr/testify/assert", funcName: containsFunc},
		{pkgPath: "github.com/stretchr/testify/require", funcName: containsFunc},
		{pkgPath: "github.com/stretchr/testify/suite", funcName: containsFunc},
	}

	if !matchesAny(call, pass.TypesInfo, containsMatchers) {
		return false
	}

	// strings.Contains first arg is the haystack; testify variants have
	// testing.T as first arg, so the haystack is the second arg.
	argIdx := errDotErrorArgIndex(call, pass)

	return argIdx >= 0 && argIdx < len(call.Args) && isErrDotError(call.Args[argIdx])
}

// errDotErrorArgIndex returns the argument index that should contain the
// haystack string for the matched function.
func errDotErrorArgIndex(call *ast.CallExpr, pass *analysis.Pass) int {
	if isCallTo(call, pass.TypesInfo, stringsPkgPath, "Contains") {
		return 0
	}

	// testify assert.Contains/require.Contains: first arg is *testing.T,
	// second arg is the collection/string to search in.
	return 1
}

// isErrDotError reports whether expr is a call to some expression's .Error()
// method with no arguments (the error interface method).
func isErrDotError(expr ast.Expr) bool {
	call, isCall := expr.(*ast.CallExpr)
	if !isCall || len(call.Args) != 0 {
		return false
	}

	sel, isSel := call.Fun.(*ast.SelectorExpr)

	return isSel && sel.Sel.Name == "Error"
}
