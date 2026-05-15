package analyzers

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// preferWGGoFanoutAnalyzer returns an analyzer that flags the pre-Go-1.25
// fan-out pattern: wg.Add(N) immediately followed by a loop that iterates
// exactly N times and spawns one goroutine per iteration starting with
// defer wg.Done(). WaitGroup.Go (Go 1.25) calls Add(1) and defers Done
// internally, so the same shape becomes one statement shorter and removes
// the chance of an Add/loop-count mismatch.
func preferWGGoFanoutAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "preferwggofanout",
		Doc:      "flags wg.Add(N) followed by a loop spawning N goroutines with defer wg.Done(); prefer wg.Go(fn) (Go 1.25)",
		Run:      runPreferWGGoFanout,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runPreferWGGoFanout(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.BlockStmt)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		block, isBlock := node.(*ast.BlockStmt)
		if !isBlock {
			return
		}

		checkBlockForWGGoFanout(block.List, pass)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkBlockForWGGoFanout scans adjacent statement pairs for wg.Add(N)
// followed by a fan-out loop. Adjacency keeps the rewrite mechanical;
// any intervening statement could rely on the Add having already happened
// before the loop starts, so we deliberately do not look past stmts[idx+1].
func checkBlockForWGGoFanout(stmts []ast.Stmt, pass *analysis.Pass) {
	for idx := range len(stmts) - 1 {
		addCall, addReceiver := extractWGAddCall(stmts[idx], pass.TypesInfo)
		if addCall == nil || len(addCall.Args) != 1 {
			continue
		}

		addCount := countExprString(addCall.Args[0], pass.TypesInfo)
		if addCount == "" {
			continue
		}

		loopCount, goStmt := extractFanoutLoop(stmts[idx+1], pass.TypesInfo)
		if loopCount == "" || goStmt == nil {
			continue
		}

		if addCount != loopCount {
			continue
		}

		if !isMatchingGoWithDeferredDone(goStmt, addReceiver, pass.TypesInfo) {
			continue
		}

		pass.Reportf(addCall.Pos(),
			"wg.Add(N) + loop spawning N goroutines with defer wg.Done() can be replaced with wg.Go(fn) (Go 1.25)")
	}
}

// extractFanoutLoop checks whether stmt is a for/range loop whose body is
// a single go statement, and returns the normalized iteration count along
// with that go statement. Returns ("", nil) for anything else, including
// loops with side-work in the body that wg.Go would re-order.
func extractFanoutLoop(stmt ast.Stmt, info *types.Info) (string, ast.Stmt) {
	switch loop := stmt.(type) {
	case *ast.RangeStmt:
		goStmt := singleGoStmtInBody(loop.Body)
		if goStmt == nil {
			return "", nil
		}

		return rangeIterCount(loop.X, info), goStmt

	case *ast.ForStmt:
		goStmt := singleGoStmtInBody(loop.Body)
		if goStmt == nil {
			return "", nil
		}

		return forIterCount(loop, info), goStmt
	}

	return "", nil
}

// singleGoStmtInBody returns the go statement when block's only statement
// is one. Loops with extra statements alongside the goroutine can't be
// migrated to wg.Go without deciding whether that work moves inside or
// stays outside the closure, which would change ordering.
func singleGoStmtInBody(block *ast.BlockStmt) ast.Stmt {
	if block == nil || len(block.List) != 1 {
		return nil
	}

	if _, isGo := block.List[0].(*ast.GoStmt); !isGo {
		return nil
	}

	return block.List[0]
}

// rangeIterCount returns the normalized iteration count of a range loop.
// Range-over-int (Go 1.22+) preserves the int expression. Range over a
// slice, array, map, string, or channel becomes len(<expr>) so that an
// Add(len(x)) on the other side compares equal without per-shape glue.
func rangeIterCount(rangeExpr ast.Expr, info *types.Info) string {
	if rangeExpr == nil {
		return ""
	}

	rangeType := info.TypeOf(rangeExpr)
	if rangeType == nil {
		return ""
	}

	if isIntegerType(rangeType) {
		return countExprString(rangeExpr, info)
	}

	inner := countExprString(rangeExpr, info)
	if inner == "" {
		return ""
	}

	return "len(" + inner + ")"
}

// forIterCount handles canonical `for i := 0; i < N; i++` loops and returns
// the normalized N. Any deviation from that exact form returns "" because
// custom init/cond/post can hide arithmetic that breaks the Add-matches-N
// equivalence the rewrite relies on.
func forIterCount(loop *ast.ForStmt, info *types.Info) string {
	if loop.Init == nil || loop.Cond == nil || loop.Post == nil {
		return ""
	}

	loopVar := extractCanonicalLoopVar(loop.Init)
	if loopVar == "" {
		return ""
	}

	boundExpr := extractCanonicalLessThanBound(loop.Cond, loopVar)
	if boundExpr == nil {
		return ""
	}

	if !isCanonicalIncrement(loop.Post, loopVar) {
		return ""
	}

	return countExprString(boundExpr, info)
}

// extractCanonicalLoopVar returns the loop variable name when init is the
// canonical `<ident> := 0`. Returns "" for any other init form, including
// multi-variable declarations or non-zero start values.
func extractCanonicalLoopVar(init ast.Stmt) string {
	assign, isAssign := init.(*ast.AssignStmt)
	if !isAssign || assign.Tok != token.DEFINE || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return ""
	}

	lhsIdent, isIdent := assign.Lhs[0].(*ast.Ident)
	if !isIdent {
		return ""
	}

	zeroLit, isLit := assign.Rhs[0].(*ast.BasicLit)
	if !isLit || zeroLit.Kind != token.INT || zeroLit.Value != "0" {
		return ""
	}

	return lhsIdent.Name
}

// extractCanonicalLessThanBound returns the right-hand side of `loopVar < N`
// when cond has that exact shape. Returns nil otherwise; `<=`, `!=`, and
// reversed orderings are intentionally rejected.
func extractCanonicalLessThanBound(cond ast.Expr, loopVar string) ast.Expr {
	binCond, isBin := cond.(*ast.BinaryExpr)
	if !isBin || binCond.Op != token.LSS {
		return nil
	}

	condVar, isVar := binCond.X.(*ast.Ident)
	if !isVar || condVar.Name != loopVar {
		return nil
	}

	return binCond.Y
}

// isCanonicalIncrement reports whether post is `loopVar++`. Other post
// forms (i += 2, decrement, multi-statement) are rejected.
func isCanonicalIncrement(post ast.Stmt, loopVar string) bool {
	incStmt, isInc := post.(*ast.IncDecStmt)
	if !isInc || incStmt.Tok != token.INC {
		return false
	}

	postIdent, isIdent := incStmt.X.(*ast.Ident)

	return isIdent && postIdent.Name == loopVar
}

// countExprString returns a normalized string form of an integer-count
// expression. Returns "" for anything that can't be reliably compared as
// a string. A missing match is preferable to a false positive when the
// rewrite would change semantics.
func countExprString(expr ast.Expr, info *types.Info) string {
	switch node := expr.(type) {
	case *ast.BasicLit:
		if node.Kind != token.INT {
			return ""
		}

		return node.Value

	case *ast.Ident:
		return node.Name

	case *ast.SelectorExpr:
		xName := countExprString(node.X, info)
		if xName == "" {
			return ""
		}

		return xName + "." + node.Sel.Name

	case *ast.CallExpr:
		ident, isIdent := node.Fun.(*ast.Ident)
		if !isIdent || ident.Name != "len" || len(node.Args) != 1 {
			return ""
		}

		if !isBuiltinLen(node, info) {
			return ""
		}

		inner := countExprString(node.Args[0], info)
		if inner == "" {
			return ""
		}

		return "len(" + inner + ")"
	}

	return ""
}

// isBuiltinLen reports whether call resolves to the builtin len. Without
// this check a user-defined function named len in scope would silently
// match, leading to false positives.
func isBuiltinLen(call *ast.CallExpr, info *types.Info) bool {
	ident, isIdent := call.Fun.(*ast.Ident)
	if !isIdent {
		return false
	}

	obj := info.ObjectOf(ident)
	if obj == nil {
		return false
	}

	_, isBuiltin := obj.(*types.Builtin)

	return isBuiltin
}

// isIntegerType reports whether t is one of Go's signed/unsigned integer
// types, including untyped int. Used to distinguish range-over-int from
// range-over-collection so the iteration count can be normalized correctly.
func isIntegerType(t types.Type) bool {
	basic, isBasic := t.Underlying().(*types.Basic)
	if !isBasic {
		return false
	}

	return basic.Info()&types.IsInteger != 0
}
