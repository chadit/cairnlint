package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// goWGGoAnalyzer returns an analyzer that flags `go wg.Go(func() {...})`.
// WaitGroup.Go (Go 1.25) calls Add(1) internally. Wrapping it with the go
// keyword makes Add(1) run in a new goroutine, racing with Wait(). This
// causes deadlocks or missed work.
func goWGGoAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "gowggo",
		Doc:      "flags go wg.Go(...); WaitGroup.Go calls Add(1) internally, wrapping with go races Add with Wait",
		Run:      runGoWGGo,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runGoWGGo(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.GoStmt)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		goStmt, isGo := node.(*ast.GoStmt)
		if !isGo {
			return
		}

		// When goStmt.Call.Fun is a *ast.FuncLit, this is `go func() { ... }()`
		// which naturally does not match a SelectorExpr. Only direct
		// `go wg.Go(...)` triggers a match.
		sel, isSel := goStmt.Call.Fun.(*ast.SelectorExpr)
		if !isSel || sel.Sel.Name != goMethodName {
			return
		}

		if !isWaitGroupReceiver(sel.X, pass.TypesInfo) && !isWaitGroupMethod(sel, pass.TypesInfo) {
			return
		}

		pass.Reportf(goStmt.Pos(), "go wg.Go(...) is a bug; wg.Go calls Add(1) internally, wrapping with go races Add with Wait")
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
