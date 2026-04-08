package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// tickerLeakAnalyzer returns an analyzer that flags time.NewTicker and
// time.NewTimer calls where the returned value is never stopped via
// defer. Tickers that aren't stopped leak a goroutine forever. Timers
// leak until expiry.
func tickerLeakAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "tickerleak",
		Doc:      "flags time.NewTicker/NewTimer without defer Stop; unstopped tickers leak a goroutine, timers leak until expiry",
		Run:      runTickerLeak,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runTickerLeak(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.AssignStmt)(nil),
	}

	insp.WithStack(nodeFilter, func(node ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		assign, isAssign := node.(*ast.AssignStmt)
		if !isAssign || len(assign.Rhs) == 0 || len(assign.Lhs) == 0 {
			return true
		}

		call, isCall := assign.Rhs[0].(*ast.CallExpr)
		if !isCall {
			return true
		}

		isTicker := isCallTo(call, pass.TypesInfo, "time", "NewTicker")
		isTimer := isCallTo(call, pass.TypesInfo, "time", "NewTimer")

		if !isTicker && !isTimer {
			return true
		}

		lhsIdent, isIdent := assign.Lhs[0].(*ast.Ident)
		if !isIdent {
			return true
		}

		varName := lhsIdent.Name

		// Walk the stack to find the enclosing block, then scan remaining
		// statements for a cleanup call or an escape hatch.
		block := findEnclosingBlock(stack)
		if block == nil {
			return true
		}

		assignIdx := stmtIndexInBlock(block, assign)
		if assignIdx < 0 {
			return true
		}

		if hasStopOrEscape(block.List[assignIdx+1:], varName) {
			return true
		}

		msg := "time.NewTimer without defer Stop leaks until expiry; add defer %s.Stop() after creation"
		if isTicker {
			msg = "time.NewTicker without defer Stop leaks a goroutine; add defer %s.Stop() after creation"
		}

		pass.Reportf(assign.Pos(), msg, varName)

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// findEnclosingBlock walks the inspector stack from innermost to outermost
// and returns the first *ast.BlockStmt that is inside a function body,
// skipping the node itself at the top of the stack.
func findEnclosingBlock(stack []ast.Node) *ast.BlockStmt {
	for idx := len(stack) - 2; idx >= 0; idx-- {
		block, isBlock := stack[idx].(*ast.BlockStmt)
		if isBlock {
			return block
		}
	}

	return nil
}

// stmtIndexInBlock returns the index of stmt within block's statement list,
// or -1 if not found.
func stmtIndexInBlock(block *ast.BlockStmt, stmt ast.Stmt) int {
	for idx, candidate := range block.List {
		if candidate == stmt {
			return idx
		}
	}

	return -1
}

// hasStopOrEscape scans the statement list for either a defer Stop call on
// varName or an escape hatch that means the caller takes responsibility.
func hasStopOrEscape(stmts []ast.Stmt, varName string) bool {
	for _, stmt := range stmts {
		// Check for defer containing varName.Stop()
		if deferHasStop(stmt, varName) {
			return true
		}

		// Escape: variable is returned, so the caller owns cleanup
		if stmtReturnsVar(stmt, varName) {
			return true
		}

		// Escape: variable assigned to a struct field (e.g., s.ticker = ticker)
		if stmtAssignsToField(stmt, varName) {
			return true
		}
	}

	return false
}

// deferHasStop reports whether stmt is a defer statement that calls
// varName.Stop(). Handles both direct defers (defer t.Stop()) and
// closure defers (defer func() { t.Stop() }()).
func deferHasStop(stmt ast.Stmt, varName string) bool {
	deferStmt, isDefer := stmt.(*ast.DeferStmt)
	if !isDefer {
		return false
	}

	// Check the entire deferred expression for a varName.Stop() call,
	// which covers both direct defer and closure wrapping.
	var found bool

	ast.Inspect(deferStmt, func(node ast.Node) bool {
		if node == nil || found {
			return false
		}

		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}

		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel || sel.Sel.Name != "Stop" {
			return true
		}

		ident, isIdent := sel.X.(*ast.Ident)
		if isIdent && ident.Name == varName {
			found = true

			return false
		}

		return true
	})

	return found
}

// stmtReturnsVar reports whether stmt is a return statement that includes
// varName as one of its results, indicating the caller owns the resource.
func stmtReturnsVar(stmt ast.Stmt, varName string) bool {
	retStmt, isRet := stmt.(*ast.ReturnStmt)
	if !isRet {
		return false
	}

	for _, result := range retStmt.Results {
		ident, isIdent := result.(*ast.Ident)
		if isIdent && ident.Name == varName {
			return true
		}
	}

	return false
}

// stmtAssignsToField reports whether stmt assigns varName to a struct field
// (e.g., s.ticker = varName), indicating the struct owns the resource.
func stmtAssignsToField(stmt ast.Stmt, varName string) bool {
	assign, isAssign := stmt.(*ast.AssignStmt)
	if !isAssign {
		return false
	}

	for idx, lhs := range assign.Lhs {
		if _, isSel := lhs.(*ast.SelectorExpr); !isSel {
			continue
		}

		if idx >= len(assign.Rhs) {
			continue
		}

		ident, isIdent := assign.Rhs[idx].(*ast.Ident)
		if isIdent && ident.Name == varName {
			return true
		}
	}

	return false
}
