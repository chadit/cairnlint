package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// queryMethods lists database method names that suggest an N+1 query pattern
// when called inside a loop body.
var queryMethods = map[string]struct{}{ //nolint:gochecknoglobals // package-internal lookup table
	"Query":           {},
	"QueryRow":        {},
	"Exec":            {},
	"QueryContext":    {},
	"QueryRowContext": {},
	"ExecContext":     {},
}

// queryInLoopAnalyzer returns an analyzer that flags database query calls
// inside loop bodies. Repeated queries per iteration are the classic N+1
// problem; batch the work or use a single query with an IN clause instead.
func queryInLoopAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "queryinloop",
		Doc:      "flags database query calls (Query, QueryRow, Exec, and their Context variants) inside for loops (N+1 pattern)",
		Run:      runQueryInLoop,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runQueryInLoop(pass *analysis.Pass) (any, error) {
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

		if !isDBQueryCall(call) {
			return true
		}

		if !isInsideLoop(stack) {
			return true
		}

		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel {
			return true
		}

		pass.Reportf(call.Pos(), "%s called inside loop; this is an N+1 query pattern, use a batch query instead", sel.Sel.Name)

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isDBQueryCall reports whether call is a method call whose name matches
// one of the known database query methods.
func isDBQueryCall(call *ast.CallExpr) bool {
	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel {
		return false
	}

	_, found := queryMethods[sel.Sel.Name]

	return found
}
