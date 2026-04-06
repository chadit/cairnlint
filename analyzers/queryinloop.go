package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// queryMethods maps database method names to the argument index where the SQL
// query string appears. Context variants take a context.Context as the first
// argument, so the query shifts to index 1.
var queryMethods = map[string]int{ //nolint:gochecknoglobals // package-internal lookup table
	"Query":           0,
	"QueryRow":        0,
	"Exec":            0,
	"QueryContext":    1,
	"QueryRowContext": 1,
	"ExecContext":     1,
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

		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel {
			return true
		}

		queryArgIdx, isDBCall := queryMethods[sel.Sel.Name]
		if !isDBCall {
			return true
		}

		if !isInsideLoop(stack) {
			return true
		}

		// When the SQL string is built with fmt.Sprintf, the query structure
		// varies per iteration (e.g., TRUNCATE on different tables). That's
		// structurally different operations, not the N+1 pattern of repeating
		// the same parameterized query with different bind values.
		if hasDynamicQueryString(call, queryArgIdx) {
			return true
		}

		pass.Reportf(call.Pos(), "%s called inside loop; this is an N+1 query pattern, use a batch query instead", sel.Sel.Name)

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// hasDynamicQueryString reports whether the SQL argument at queryArgIdx is
// built with fmt.Sprintf. When the query itself is formatted per iteration
// (e.g., different table names), the operations are structurally different
// and not the N+1 anti-pattern.
func hasDynamicQueryString(call *ast.CallExpr, queryArgIdx int) bool {
	if queryArgIdx >= len(call.Args) {
		return false
	}

	inner, isCall := call.Args[queryArgIdx].(*ast.CallExpr)
	if !isCall {
		return false
	}

	return isFmtSprintf(inner)
}
