package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// typeAssertNoCheckAnalyzer returns an analyzer that flags single-value type
// assertions like x := y.(Type). Without the comma-ok form, a failed
// assertion panics at runtime. Use x, ok := y.(Type) or a type switch.
func typeAssertNoCheckAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "typeassertnocheck",
		Doc:      "flags single-value type assertions y.(Type); use the comma-ok form or a type switch to avoid panics",
		Run:      runTypeAssertNoCheck,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runTypeAssertNoCheck(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.AssignStmt)(nil),
		(*ast.ExprStmt)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		switch stmt := node.(type) {
		case *ast.AssignStmt:
			checkAssignTypeAssert(pass, stmt)
		case *ast.ExprStmt:
			// A bare type assertion as a statement (rare but valid).
			if assert, isAssert := stmt.X.(*ast.TypeAssertExpr); isAssert {
				if assert.Type != nil {
					pass.Reportf(assert.Pos(), "unchecked type assertion; use the comma-ok form x, ok := y.(Type) to avoid panics")
				}
			}
		}
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkAssignTypeAssert flags assignment statements where a type assertion
// produces only one value (no ok variable).
func checkAssignTypeAssert(pass *analysis.Pass, stmt *ast.AssignStmt) {
	if len(stmt.Rhs) != 1 {
		return
	}

	assert, isAssert := stmt.Rhs[0].(*ast.TypeAssertExpr)
	if !isAssert || assert.Type == nil {
		return
	}

	// The comma-ok form produces two LHS values. A single LHS means the
	// assertion is unchecked and will panic on failure.
	if len(stmt.Lhs) == 1 {
		pass.Reportf(assert.Pos(), "unchecked type assertion; use the comma-ok form x, ok := y.(Type) to avoid panics")
	}
}
