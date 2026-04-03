package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// contextQueryMethods lists database methods that accept a context as their
// first argument and should not receive a bare context.Background().
var contextQueryMethods = map[string]struct{}{ //nolint:gochecknoglobals // package-internal lookup table
	"QueryContext":    {},
	"QueryRowContext": {},
	"ExecContext":     {},
}

// dbQueryWithBareBackgroundAnalyzer returns an analyzer that flags database
// context-aware query methods called with context.Background() as the first
// argument. Production code should propagate a request-scoped context for
// cancellation and tracing support.
func dbQueryWithBareBackgroundAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "dbquerywithbarebackground",
		Doc:      "flags db.QueryContext/ExecContext/QueryRowContext with context.Background(); pass a request-scoped context instead",
		Run:      runDBQueryWithBareBackground,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runDBQueryWithBareBackground(pass *analysis.Pass) (any, error) {
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

		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel {
			return
		}

		_, isContextQuery := contextQueryMethods[sel.Sel.Name]
		if !isContextQuery {
			return
		}

		if !hasBackgroundArg(call, pass) {
			return
		}

		pass.Reportf(call.Pos(), "%s called with context.Background(); pass a request-scoped context instead", sel.Sel.Name)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
