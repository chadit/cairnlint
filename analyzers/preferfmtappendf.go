package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// preferFmtAppendfAnalyzer returns an analyzer that flags []byte(fmt.Sprintf(...))
// conversions. fmt.Appendf(nil, ...) writes directly into a byte slice and
// avoids the intermediate string allocation that the conversion requires.
func preferFmtAppendfAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "preferfmtappendf",
		Doc:      "flags []byte(fmt.Sprintf(...)); use fmt.Appendf(nil, ...) to avoid intermediate string allocation",
		Run:      runPreferFmtAppendf,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runPreferFmtAppendf(pass *analysis.Pass) (any, error) {
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

		if !isByteSliceConversion(call) {
			return
		}

		if len(call.Args) != 1 {
			return
		}

		innerCall, isInnerCall := call.Args[0].(*ast.CallExpr)
		if !isInnerCall {
			return
		}

		if !isCallTo(innerCall, pass.TypesInfo, "fmt", "Sprintf") {
			return
		}

		pass.Reportf(call.Pos(), "use fmt.Appendf(nil, ...) instead of []byte(fmt.Sprintf(...)) to avoid intermediate string allocation")
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isByteSliceConversion reports whether call is a type conversion to []byte.
// The AST represents []byte(x) as a CallExpr where Fun is an ArrayType with
// nil Len and a byte Elt.
func isByteSliceConversion(call *ast.CallExpr) bool {
	arrayType, isArray := call.Fun.(*ast.ArrayType)
	if !isArray || arrayType.Len != nil {
		return false
	}

	eltIdent, isIdent := arrayType.Elt.(*ast.Ident)

	return isIdent && eltIdent.Name == "byte"
}
