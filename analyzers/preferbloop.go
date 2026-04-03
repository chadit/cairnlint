package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const bLoopMessage = "use b.Loop() { ... } instead of manual b.N loop (Go 1.24+)"

// preferBLoopAnalyzer returns an analyzer that flags old-style benchmark loops
// using b.N in test files. Go 1.24 introduced b.Loop() which handles
// iteration counting, timer resets, and prevents compiler dead-code
// elimination automatically.
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

	insp.Preorder(nodeFilter, func(node ast.Node) {
		if !isTestFile(pass, node) {
			return
		}

		switch stmt := node.(type) {
		case *ast.ForStmt:
			if isCStyleBNLoop(stmt, pass.TypesInfo) {
				pass.Reportf(stmt.Pos(), "%s", bLoopMessage)
			}
		case *ast.RangeStmt:
			if isRangeBN(stmt, pass.TypesInfo) {
				pass.Reportf(stmt.Pos(), "%s", bLoopMessage)
			}
		}
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
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

	recvType := info.TypeOf(sel.X)
	if recvType == nil {
		return false
	}

	ptr, isPtr := recvType.(*types.Pointer)
	if !isPtr {
		return false
	}

	named, isNamed := ptr.Elem().(*types.Named)
	if !isNamed {
		return false
	}

	obj := named.Obj()

	return obj.Pkg() != nil && obj.Pkg().Path() == "testing" && obj.Name() == "B"
}
