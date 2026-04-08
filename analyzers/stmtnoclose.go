package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// stmtNoCloseAnalyzer returns an analyzer that flags db.Prepare and
// db.PrepareContext calls where the returned *sql.Stmt is never closed
// via defer. Unclosed prepared statements leak server-side resources
// and file descriptors.
func stmtNoCloseAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "stmtnoclose",
		Doc:      "flags db.Prepare/PrepareContext without defer Close; unclosed prepared statements leak server-side resources",
		Run:      runStmtNoClose,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runStmtNoClose(pass *analysis.Pass) (any, error) {
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

		// Match db.Prepare or db.PrepareContext where the receiver is
		// from database/sql (covers *sql.DB and *sql.Tx).
		isPrepare := isCallTo(call, pass.TypesInfo, "database/sql", "Prepare")
		isPrepareCtx := isCallTo(call, pass.TypesInfo, "database/sql", "PrepareContext")

		if !isPrepare && !isPrepareCtx {
			return true
		}

		lhsIdent, isIdent := assign.Lhs[0].(*ast.Ident)
		if !isIdent {
			return true
		}

		varName := lhsIdent.Name

		block := findEnclosingBlock(stack)
		if block == nil {
			return true
		}

		assignIdx := stmtIndexInBlock(block, assign)
		if assignIdx < 0 {
			return true
		}

		if hasCloseOrEscape(block.List[assignIdx+1:], varName) {
			return true
		}

		pass.Reportf(assign.Pos(), "db.Prepare without defer Close leaks prepared statements; add defer %s.Close() after error check", varName)

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// hasCloseOrEscape scans the statement list for either a defer Close call on
// varName or an escape hatch that means the caller takes responsibility.
func hasCloseOrEscape(stmts []ast.Stmt, varName string) bool {
	for _, stmt := range stmts {
		if deferHasClose(stmt, varName) {
			return true
		}

		// Escape: variable is returned, so the caller owns cleanup.
		if stmtReturnsVar(stmt, varName) {
			return true
		}

		// Escape: variable assigned to a struct field (e.g., s.stmt = stmt).
		if stmtAssignsToField(stmt, varName) {
			return true
		}
	}

	return false
}

// deferHasClose reports whether stmt is a defer statement that calls
// varName.Close(). Handles both direct defers (defer stmt.Close()) and
// closure defers (defer func() { stmt.Close() }()).
func deferHasClose(stmt ast.Stmt, varName string) bool {
	deferStmt, isDefer := stmt.(*ast.DeferStmt)
	if !isDefer {
		return false
	}

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
		if !isSel || sel.Sel.Name != "Close" {
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
