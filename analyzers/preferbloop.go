package analyzers

import (
	"go/ast"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const bLoopMessage = "use b.Loop() { ... } instead of manual b.N loop (Go 1.24+)"

// Benchmark timer controls that b.Loop supersedes. A benchmark calling either
// one cannot be converted mechanically, so their presence suppresses the
// diagnostic.
const (
	benchStopTimer  = "StopTimer"
	benchStartTimer = "StartTimer"
)

// preferBLoopAnalyzer returns an analyzer that flags old-style benchmark loops
// using b.N in test files. Go 1.24 introduced b.Loop() which handles
// iteration counting, timer resets, and prevents compiler dead-code
// elimination automatically.
//
// Benchmarks that drive the timer by hand are left alone. b.Loop keeps its own
// timing window, so rewriting a loop whose enclosing function calls
// b.StopTimer or b.StartTimer changes what the benchmark measures and can hang
// it outright (golang/go#74967). Upstream disabled its own bloop modernizer
// over this; the rule is sound, the blanket rewrite was not.
//
// A benchmark that reads b.N anywhere other than the one loop condition is
// also left alone, as upstream's usesBenchmarkNOnce does. Under b.Loop the
// runner sets b.N to 1 and then 0, so a slice sized by b.N is indexed out of
// range and a second b.N loop fails with "B.Loop called with timer stopped".
// For the same reason a loop inside a closure that captures an outer b is
// skipped: the outer body may size data by b.N where the closure cannot see.
func preferBLoopAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "preferbloop",
		Doc:      "flags for i := 0; i < b.N; i++ and for range b.N in benchmarks; use b.Loop() instead (Go 1.24+)",
		Run:      runPreferBLoop,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runPreferBLoop(pass *analysis.Pass) (any, error) {
	if !hasTestFiles(pass) {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.ForStmt)(nil),
		(*ast.RangeStmt)(nil),
	}

	insp.WithStack(nodeFilter, func(node ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		if !isTestFile(pass, node) || !isBNLoop(node, pass.TypesInfo) {
			return true
		}

		if !goVersionAtLeast(pass, node.Pos(), goVersion124) {
			return true
		}

		if !convertibleToBLoop(node, stack, pass.TypesInfo) {
			return true
		}

		pass.Reportf(node.Pos(), "%s", bLoopMessage)

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isBNLoop reports whether node is either loop form driven by b.N.
func isBNLoop(node ast.Node, info *types.Info) bool {
	switch stmt := node.(type) {
	case *ast.ForStmt:
		return isCStyleBNLoop(stmt, info)
	case *ast.RangeStmt:
		return isRangeBN(stmt, info)
	default:
		return false
	}
}

// isCStyleBNLoop reports whether stmt looks like: for i := 0; i < b.N; i++
// where b is of type *testing.B.
func isCStyleBNLoop(stmt *ast.ForStmt, info *types.Info) bool {
	if stmt.Cond == nil {
		return false
	}

	binExpr, isBin := stmt.Cond.(*ast.BinaryExpr)
	if !isBin {
		return false
	}

	return isBNSelector(binExpr.Y, info)
}

// isRangeBN reports whether stmt looks like: for range b.N
// where b is of type *testing.B.
func isRangeBN(stmt *ast.RangeStmt, info *types.Info) bool {
	if stmt.X == nil {
		return false
	}

	return isBNSelector(stmt.X, info)
}

// isBNSelector reports whether expr is a selector expression b.N where b
// resolves to *testing.B.
func isBNSelector(expr ast.Expr, info *types.Info) bool {
	sel, isSel := expr.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != "N" {
		return false
	}

	return isTestingBExpr(sel.X, info)
}

// convertibleToBLoop reports whether the b.N loop on node can become b.Loop.
// The loop's b must be a parameter of the innermost enclosing function, that
// function must not drive the timer by hand, and b.N must appear nowhere
// else in its body.
//
// Innermost is the right scope because each b.Run sub-benchmark receives its
// own *testing.B: timer calls and b.N reads in an outer benchmark say nothing
// about a loop inside a sub-benchmark closure. Requiring b to be that
// function's own parameter is what tells a sub-benchmark apart from a
// closure that captures the outer b.
func convertibleToBLoop(node ast.Node, stack []ast.Node, info *types.Info) bool {
	funcType, body := enclosingFunc(stack)
	if body == nil {
		return false
	}

	benchObj := info.ObjectOf(bnLoopReceiver(node))
	if benchObj == nil || !declaresParam(funcType, info, benchObj) {
		return false
	}

	return !callsBenchTimerControl(body, info) && countBNReferences(body, info) == 1
}

// bnLoopReceiver returns the identifier before .N in a b.N loop already
// accepted by isBNLoop, or nil when that operand is not a plain identifier.
func bnLoopReceiver(node ast.Node) *ast.Ident {
	var operand ast.Expr

	switch stmt := node.(type) {
	case *ast.ForStmt:
		if bin, isBin := stmt.Cond.(*ast.BinaryExpr); isBin {
			operand = bin.Y
		}
	case *ast.RangeStmt:
		operand = stmt.X
	}

	sel, isSel := operand.(*ast.SelectorExpr)
	if !isSel {
		return nil
	}

	ident, _ := sel.X.(*ast.Ident)

	return ident
}

// declaresParam reports whether obj is one of funcType's parameters.
func declaresParam(funcType *ast.FuncType, info *types.Info, obj types.Object) bool {
	if funcType.Params == nil {
		return false
	}

	for _, field := range funcType.Params.List {
		for _, name := range field.Names {
			if info.Defs[name] == obj {
				return true
			}
		}
	}

	return false
}

// enclosingFunc returns the signature and body of the innermost function
// surrounding the node on top of stack.
func enclosingFunc(stack []ast.Node) (*ast.FuncType, *ast.BlockStmt) {
	for idx := range slices.Backward(stack) {
		switch fn := stack[idx].(type) {
		case *ast.FuncDecl:
			return fn.Type, fn.Body
		case *ast.FuncLit:
			return fn.Type, fn.Body
		}
	}

	return nil, nil
}

// callsBenchTimerControl reports whether body contains a StopTimer or
// StartTimer call on a *testing.B.
func callsBenchTimerControl(body *ast.BlockStmt, info *types.Info) bool {
	if body == nil {
		return false
	}

	var found bool

	ast.Inspect(body, func(node ast.Node) bool {
		if found {
			return false
		}

		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}

		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel {
			return true
		}

		if sel.Sel.Name != benchStopTimer && sel.Sel.Name != benchStartTimer {
			return true
		}

		found = isTestingBExpr(sel.X, info)

		return !found
	})

	return found
}

// countBNReferences counts b.N selectors in body. The loop under review
// contributes one. Any other reference means b.N carries meaning that
// b.Loop cannot preserve.
func countBNReferences(body *ast.BlockStmt, info *types.Info) int {
	if body == nil {
		return 0
	}

	var count int

	ast.Inspect(body, func(node ast.Node) bool {
		if expr, isExpr := node.(ast.Expr); isExpr && isBNSelector(expr, info) {
			count++
		}

		return true
	})

	return count
}
