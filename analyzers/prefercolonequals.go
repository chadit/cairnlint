package analyzers

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// preferColonEqualsAnalyzer returns an analyzer that flags a function-scoped
// var declaration that initializes a single name to a non-zero value. The
// short form (x := v) is the convention for non-zero initialization. This pairs
// with the zero-value rule, which keeps var for the empty-and-ready case.
func preferColonEqualsAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "prefercolonequals",
		Doc:      "flags var x = v with a non-zero value inside functions; use x := v",
		Run:      runPreferColonEquals,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runPreferColonEquals(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.DeclStmt)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		declStmt, isDecl := node.(*ast.DeclStmt)
		if !isDecl {
			return
		}

		genDecl, isGen := declStmt.Decl.(*ast.GenDecl)
		if !isGen || genDecl.Tok != token.VAR {
			return
		}

		// A parenthesized var ( ... ) block is a deliberate grouping; leave it.
		if genDecl.Lparen.IsValid() || len(genDecl.Specs) != 1 {
			return
		}

		spec, isValue := genDecl.Specs[0].(*ast.ValueSpec)
		if !isValue {
			return
		}

		// Skip explicit types (var x int = f()), multi-name specs, the
		// zero-value form (var x int), and the blank identifier.
		if spec.Type != nil || len(spec.Names) != 1 || len(spec.Values) != 1 {
			return
		}

		if spec.Names[0].Name == "_" || isZeroValueExpr(spec.Values[0]) {
			return
		}

		pass.Reportf(genDecl.Pos(), "use %s := ... instead of var %s = ... for non-zero initialization", spec.Names[0].Name, spec.Names[0].Name)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isZeroValueExpr reports whether expr is a literal zero value. Those are left
// alone so this rule does not fight the zero-value declaration convention.
func isZeroValueExpr(expr ast.Expr) bool {
	switch value := expr.(type) {
	case *ast.BasicLit:
		return isZeroBasicLit(value)
	case *ast.Ident:
		return value.Name == "false" || value.Name == "nil"
	}

	return false
}

// isZeroBasicLit reports whether lit is the zero literal for its kind.
func isZeroBasicLit(lit *ast.BasicLit) bool {
	if lit.Kind == token.INT {
		return lit.Value == "0"
	}

	if lit.Kind == token.FLOAT {
		return lit.Value == "0.0"
	}

	if lit.Kind == token.STRING {
		return lit.Value == `""` || lit.Value == "``"
	}

	return false
}
