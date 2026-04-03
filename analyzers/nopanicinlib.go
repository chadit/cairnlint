package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// noPanicInLibAnalyzer returns an analyzer that flags panic() calls in
// non-test files. Library code should return errors instead of panicking,
// because panics crash the caller with no opportunity to recover gracefully.
func noPanicInLibAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "nopanicinlib",
		Doc:      "flags panic() in non-test files; library code should return errors",
		Run:      runNoPanicInLib,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runNoPanicInLib(pass *analysis.Pass) (any, error) {
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

		// Skip test files where panic is acceptable for test helpers.
		if isTestFile(pass, call) {
			return
		}

		ident, isIdent := call.Fun.(*ast.Ident)
		if !isIdent || ident.Name != "panic" {
			return
		}

		pass.Reportf(call.Pos(), "panic() in library code; return an error instead")
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
