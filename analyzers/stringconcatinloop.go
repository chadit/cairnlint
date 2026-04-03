package analyzers

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// stringConcatInLoopAnalyzer returns an analyzer that flags string
// concatenation (+=) inside loop bodies. Each concatenation allocates a new
// string, turning an O(n) operation into O(n^2). Use strings.Builder instead.
func stringConcatInLoopAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "stringconcatinloop",
		Doc:      "flags string concatenation with += inside for loops; use strings.Builder instead",
		Run:      runStringConcatInLoop,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runStringConcatInLoop(pass *analysis.Pass) (any, error) {
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
		if !isAssign {
			return true
		}

		if !isStringPlusAssign(assign, pass.TypesInfo) {
			return true
		}

		if !isInsideLoop(stack) {
			return true
		}

		pass.Reportf(assign.Pos(), "string concatenation with += inside loop; use strings.Builder instead")

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isStringPlusAssign reports whether assign is a string concatenation
// assignment: either `s += x` or `s = s + x` where s is a string.
func isStringPlusAssign(assign *ast.AssignStmt, info *types.Info) bool {
	if len(assign.Lhs) == 0 || len(assign.Rhs) == 0 {
		return false
	}

	if !isStringType(assign.Lhs[0], info) {
		return false
	}

	// Direct += operator.
	if assign.Tok == token.ADD_ASSIGN {
		return true
	}

	// Explicit s = s + x form.
	if assign.Tok != token.ASSIGN {
		return false
	}

	binExpr, isBin := assign.Rhs[0].(*ast.BinaryExpr)
	if !isBin || binExpr.Op != token.ADD {
		return false
	}

	lhsIdent, lhsIsIdent := assign.Lhs[0].(*ast.Ident)
	rhsIdent, rhsIsIdent := binExpr.X.(*ast.Ident)

	return lhsIsIdent && rhsIsIdent && lhsIdent.Obj == rhsIdent.Obj
}

// isStringType reports whether expr has an underlying string type.
func isStringType(expr ast.Expr, info *types.Info) bool {
	exprType := info.TypeOf(expr)
	if exprType == nil {
		return false
	}

	basic, isBasic := exprType.Underlying().(*types.Basic)

	return isBasic && basic.Kind() == types.String
}
