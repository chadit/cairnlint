package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// deferInLoopAnalyzer returns an analyzer that flags defer statements placed
// inside loop bodies. Deferred calls stack until the enclosing function
// returns rather than per-iteration, so resources accumulate and leak.
func deferInLoopAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "deferinloop",
		Doc:      "flags defer statements inside for loops; deferred calls stack until the function returns, causing resource leaks",
		Run:      runDeferInLoop,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runDeferInLoop(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.DeferStmt)(nil),
	}

	insp.WithStack(nodeFilter, func(node ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		if !isInsideLoop(stack) {
			return true
		}

		pass.Reportf(node.Pos(), "defer inside loop; deferred calls execute when the function returns, not per iteration")

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isInsideLoop walks the stack (excluding the node itself at the end) and
// returns true if any ancestor is a for/range loop. Stops at function
// boundaries so a deferred call inside a closure inside a loop is not flagged.
func isInsideLoop(stack []ast.Node) bool {
	// Walk from just below the current node (len-2) up to the root.
	// Stop at FuncLit/FuncDecl boundaries because the deferred call belongs
	// to that inner function, not to the loop's enclosing function.
	for idx := len(stack) - 2; idx >= 0; idx-- {
		switch stack[idx].(type) {
		case *ast.FuncLit:
			return false
		case *ast.FuncDecl:
			return false
		case *ast.ForStmt, *ast.RangeStmt:
			return true
		}
	}

	return false
}
