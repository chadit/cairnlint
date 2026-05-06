package analyzers

import (
	"go/ast"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// reflectNoKindCheckAnalyzer returns an analyzer that flags reflect.Type.Fields(),
// reflect.Type.NumField(), reflect.Value.Fields(), and reflect.Value.NumField()
// calls without a preceding Kind check. All four methods panic if
// Kind() != reflect.Struct.
func reflectNoKindCheckAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "reflectnokindcheck",
		Doc:      "flags reflect Fields/NumField calls without a preceding Kind check; both methods panic if Kind is not Struct",
		Run:      runReflectNoKindCheck,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runReflectNoKindCheck(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.WithStack(nodeFilter, func(node ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}

		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel {
			return true
		}

		methodName := sel.Sel.Name
		if methodName != "Fields" && methodName != "NumField" {
			return true
		}

		if !isReflectMethod(sel, pass.TypesInfo) {
			return true
		}

		receiver := receiverIdent(sel.X)
		if receiver == "" {
			return true
		}

		if isKindGuarded(stack, receiver) {
			return true
		}

		pass.Reportf(
			call.Pos(),
			"reflect.Type.%s() panics if Kind is not Struct; add a Kind check before calling %s",
			methodName, methodName,
		)

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isReflectMethod reports whether the selector refers to a method declared in
// the reflect package. This catches methods on both reflect.Type (interface)
// and reflect.Value (struct).
func isReflectMethod(sel *ast.SelectorExpr, info *types.Info) bool {
	obj := info.ObjectOf(sel.Sel)
	if obj == nil {
		return false
	}

	pkg := obj.Pkg()

	return pkg != nil && pkg.Path() == reflectPkgPath
}

// isKindGuarded walks the AST stack backwards from the call site looking for
// an enclosing if-statement or switch-statement that checks .Kind() on the
// same receiver. Stops at function boundaries.
func isKindGuarded(stack []ast.Node, receiver string) bool {
	for idx := range slices.Backward(stack) {
		switch node := stack[idx].(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			// Reached a function boundary without finding a guard.
			return false

		case *ast.IfStmt:
			if condContainsKindCall(node.Cond, receiver) {
				return true
			}

		case *ast.CaseClause:
			// Walk further back to find the parent SwitchStmt whose body
			// contains this CaseClause.
			parentSwitch := findParentSwitchStmt(stack, idx)
			if parentSwitch != nil && tagCallsKind(parentSwitch.Tag, receiver) {
				return true
			}
		}
	}

	return false
}

// condContainsKindCall reports whether cond contains a call to .Kind() on the
// specified receiver. Uses ast.Inspect to handle compound conditions like
// t.Kind() == reflect.Struct && someOtherCheck.
func condContainsKindCall(cond ast.Expr, receiver string) bool {
	var found bool

	ast.Inspect(cond, func(node ast.Node) bool {
		if node == nil || found {
			return false
		}

		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}

		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel || sel.Sel.Name != "Kind" {
			return true
		}

		if receiverIdent(sel.X) == receiver {
			found = true

			return false
		}

		return true
	})

	return found
}

// tagCallsKind reports whether the switch tag expression is a call to .Kind()
// on the specified receiver.
func tagCallsKind(tag ast.Expr, receiver string) bool {
	if tag == nil {
		return false
	}

	call, isCall := tag.(*ast.CallExpr)
	if !isCall {
		return false
	}

	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != "Kind" {
		return false
	}

	return receiverIdent(sel.X) == receiver
}

// findParentSwitchStmt walks the stack backwards from idx looking for the
// *ast.SwitchStmt that contains the CaseClause at stack[idx]. The switch
// statement is typically two positions back: stack has [..., SwitchStmt,
// BlockStmt (the Body), CaseClause, ...].
func findParentSwitchStmt(stack []ast.Node, caseIdx int) *ast.SwitchStmt {
	for searchIdx := caseIdx - 1; searchIdx >= 0; searchIdx-- {
		if switchStmt, isSwitch := stack[searchIdx].(*ast.SwitchStmt); isSwitch {
			return switchStmt
		}
	}

	return nil
}
