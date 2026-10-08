package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const benchResetTimerMessage = "benchmark has setup code without b.ResetTimer(); setup time is included in measurements"

// benchResetTimerAnalyzer returns an analyzer that flags benchmark functions
// with setup code before a b.N loop but no b.ResetTimer() call. Without
// ResetTimer the setup cost pollutes the measurement. Benchmarks that use
// for b.Loop() are exempt: Loop resets the timer itself on its first call
// (Go 1.24+), so setup before it is never measured.
func benchResetTimerAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "benchresettimer",
		Doc:      "flags benchmarks with setup code before a b.N loop but no b.ResetTimer()",
		Run:      runBenchResetTimer,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runBenchResetTimer(pass *analysis.Pass) (any, error) {
	if !hasTestFiles(pass) {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		funcDecl, isFuncDecl := node.(*ast.FuncDecl)
		if !isFuncDecl {
			return
		}

		if !isTestFile(pass, funcDecl) {
			return
		}

		if !isBenchmarkFunc(funcDecl, pass.TypesInfo) {
			return
		}

		checkBenchBody(pass, funcDecl)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkBenchBody looks for a b.N loop in the function body, checks whether
// setup statements precede the loop, and whether b.ResetTimer() is called
// between setup and the loop. A b.Loop() benchmark never matches, because
// Loop resets the timer itself.
func checkBenchBody(pass *analysis.Pass, funcDecl *ast.FuncDecl) {
	if funcDecl.Body == nil || len(funcDecl.Body.List) == 0 {
		return
	}

	loopIdx := findBNLoopIndex(funcDecl.Body.List, pass.TypesInfo)
	if loopIdx < 0 {
		return
	}

	// No statements before the loop means no setup to worry about.
	if loopIdx == 0 {
		return
	}

	// Check whether any statement between the start and the loop is a
	// b.ResetTimer() call.
	var found bool

	for idx := range loopIdx {
		if isResetTimerCall(funcDecl.Body.List[idx], pass.TypesInfo) {
			found = true

			break
		}
	}

	if !found {
		pass.Reportf(funcDecl.Name.Pos(), "%s", benchResetTimerMessage)
	}
}

// findBNLoopIndex returns the index of the first for statement in stmts
// that iterates using b.N, in either the C-style or range form. Returns -1
// if none found.
func findBNLoopIndex(stmts []ast.Stmt, info *types.Info) int {
	for idx, stmt := range stmts {
		switch loopStmt := stmt.(type) {
		case *ast.ForStmt:
			if isCStyleBNCondition(loopStmt.Cond, info) {
				return idx
			}
		case *ast.RangeStmt:
			if isBNSelector(loopStmt.X, info) {
				return idx
			}
		}
	}

	return -1
}

// isCStyleBNCondition reports whether expr is a binary comparison involving
// b.N (e.g. i < b.N).
func isCStyleBNCondition(expr ast.Expr, info *types.Info) bool {
	bin, isBin := expr.(*ast.BinaryExpr)
	if !isBin {
		return false
	}

	return isBNSelector(bin.Y, info)
}

// isResetTimerCall reports whether stmt is an expression statement calling
// b.ResetTimer() where b is *testing.B.
func isResetTimerCall(stmt ast.Stmt, info *types.Info) bool {
	exprStmt, isExprStmt := stmt.(*ast.ExprStmt)
	if !isExprStmt {
		return false
	}

	call, isCall := exprStmt.X.(*ast.CallExpr)
	if !isCall {
		return false
	}

	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != "ResetTimer" {
		return false
	}

	return isTestingBExpr(sel.X, info)
}
