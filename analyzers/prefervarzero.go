package analyzers

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// preferVarZeroAnalyzer returns an analyzer that flags short variable
// declarations that assign a zero value literal. Using var declarations
// (var s string, var n int, var b bool) makes the zero-value intent
// explicit and is the idiomatic Go style.
func preferVarZeroAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "prefervarzero",
		Doc:      `flags s := "", n := 0, b := false; use var s string, var n int, var b bool for zero-value init`,
		Run:      runPreferVarZero,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runPreferVarZero(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.AssignStmt)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		assign, isAssign := node.(*ast.AssignStmt)
		if !isAssign || assign.Tok != token.DEFINE {
			return
		}

		// Only flag single-variable short declarations.
		if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return
		}

		suggestion := zeroValueSuggestion(assign.Rhs[0])
		if suggestion == "" {
			return
		}

		ident, isIdent := assign.Lhs[0].(*ast.Ident)
		if !isIdent {
			return
		}

		pass.Reportf(assign.Pos(), "use \"var %s %s\" instead of short declaration with zero value", ident.Name, suggestion)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// zeroValueSuggestion returns the var type name if expr is a zero-value
// literal ("", 0, false), or empty string if it is not.
func zeroValueSuggestion(expr ast.Expr) string {
	lit, isLit := expr.(*ast.BasicLit)
	if isLit {
		switch {
		case lit.Kind == token.STRING && lit.Value == `""`:
			return "string"
		case lit.Kind == token.INT && lit.Value == "0":
			return "int"
		case lit.Kind == token.FLOAT && lit.Value == "0.0":
			return "float64"
		}

		return ""
	}

	// false is an *ast.Ident, not a BasicLit.
	ident, isIdent := expr.(*ast.Ident)
	if isIdent && ident.Name == "false" {
		return "bool"
	}

	return ""
}
