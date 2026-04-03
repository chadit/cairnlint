package analyzers

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// discardedContextAnalyzer returns an analyzer that flags function parameters
// declared as `_ context.Context`. Discarding a context breaks the
// cancellation chain and prevents deadline propagation.
func discardedContextAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "discardedcontext",
		Doc:  "flags _ context.Context parameters; discarding context breaks cancellation propagation",
		Run:  runDiscardedContext,
	}
}

func runDiscardedContext(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename

		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		if strings.Contains(filename, "test/mocks/") {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.FuncDecl:
				checkParamsForDiscardedContext(pass, typed.Type.Params)
			case *ast.FuncLit:
				checkParamsForDiscardedContext(pass, typed.Type.Params)
			}

			return true
		})
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkParamsForDiscardedContext walks a parameter list looking for
// `_ context.Context` entries and reports each one.
func checkParamsForDiscardedContext(pass *analysis.Pass, params *ast.FieldList) {
	if params == nil {
		return
	}

	for _, field := range params.List {
		if !isContextContextType(field.Type) {
			continue
		}

		for _, name := range field.Names {
			if name.Name == "_" {
				pass.Reportf(name.Pos(), "discarded context.Context breaks cancellation; use the parameter or remove it")
			}
		}
	}
}

// isContextContextType reports whether expr refers to context.Context
// using AST-level inspection (selector expression "context.Context").
func isContextContextType(expr ast.Expr) bool {
	sel, isSel := expr.(*ast.SelectorExpr)
	if !isSel {
		return false
	}

	ident, isIdent := sel.X.(*ast.Ident)
	if !isIdent {
		return false
	}

	return ident.Name == "context" && sel.Sel.Name == "Context"
}
