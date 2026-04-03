package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// preferErrorsAsTypeAnalyzer returns an analyzer that flags errors.As(err, &target)
// calls. Go 1.26 introduced errors.AsType[T](err) which is type-safe and
// eliminates the need for a pre-declared pointer variable.
func preferErrorsAsTypeAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "prefererrorsastype",
		Doc:      "flags errors.As(); use errors.AsType[T]() (Go 1.26+) for type-safe error unwrapping",
		Run:      runPreferErrorsAsType,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runPreferErrorsAsType(pass *analysis.Pass) (any, error) {
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

		if !isCallTo(call, pass.TypesInfo, "errors", "As") {
			return
		}

		pass.Reportf(call.Pos(), "use errors.AsType[T](err) instead of errors.As(err, &target) for type-safe error unwrapping")
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
