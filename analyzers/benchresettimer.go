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
// with setup code before the benchmark loop but no b.ResetTimer() call.
// Without ResetTimer the setup cost pollutes the measurement.
func benchResetTimerAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "benchresettimer",
		Doc:      "flags benchmarks with setup code before the loop but no b.ResetTimer()",
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

// checkBenchBody looks for a benchmark loop in the function body, checks
// whether setup statements precede the loop, and whether b.ResetTimer()
// is called between setup and the loop.
func checkBenchBody(pass *analysis.Pass, funcDecl *ast.FuncDecl) {
	if funcDecl.Body == nil || len(funcDecl.Body.List) == 0 {
		return
	}

	loopIdx := findBenchLoopIndex(funcDecl.Body.List, pass.TypesInfo)
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

// findBenchLoopIndex returns the index of the first for statement in stmts
// that iterates using b.Loop() or b.N. Returns -1 if none found.
func findBenchLoopIndex(stmts []ast.Stmt, info *types.Info) int {
	for idx, stmt := range stmts {
		switch loopStmt := stmt.(type) {
		case *ast.ForStmt:
			if isBLoopCall(loopStmt.Cond, info) || isCStyleBNCondition(loopStmt.Cond, info) {
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

// isBLoopCall reports whether expr is a call to b.Loop() where b is *testing.B.
func isBLoopCall(expr ast.Expr, info *types.Info) bool {
	call, isCall := expr.(*ast.CallExpr)
	if !isCall {
		return false
	}

	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != "Loop" {
		return false
	}

	return isTestingBReceiver(sel.X, info)
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

// isTestingBReceiver reports whether expr resolves to a *testing.B value.
func isTestingBReceiver(expr ast.Expr, info *types.Info) bool {
	exprType := info.TypeOf(expr)
	if exprType == nil {
		return false
	}

	ptr, isPtr := exprType.(*types.Pointer)
	if !isPtr {
		return false
	}

	named, isNamed := ptr.Elem().(*types.Named)
	if !isNamed {
		return false
	}

	obj := named.Obj()

	return obj.Pkg() != nil && obj.Pkg().Path() == testingPkgPath && obj.Name() == "B"
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

	return isTestingBReceiver(sel.X, info)
}
