package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// noContextInStructAnalyzer returns an analyzer that flags context.Context
// stored as a struct field. Per the Go documentation, context should be
// passed as the first function parameter, not embedded in structs, because
// struct lifetimes rarely match request lifetimes.
func noContextInStructAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "nocontextinstruct",
		Doc:      "flags context.Context as a struct field; pass context as a function parameter instead",
		Run:      runNoContextInStruct,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runNoContextInStruct(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.StructType)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		structType, isStruct := node.(*ast.StructType)
		if !isStruct || structType.Fields == nil {
			return
		}

		for _, field := range structType.Fields.List {
			if isContextType(field.Type) {
				pass.Reportf(field.Pos(), "context.Context should not be stored in a struct; pass it as a function parameter")
			}
		}
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isContextType reports whether expr refers to context.Context.
// Handles both qualified (context.Context) and unqualified (Context after
// dot-import) forms.
func isContextType(expr ast.Expr) bool {
	sel, isSel := expr.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != "Context" {
		return false
	}

	ident, isIdent := sel.X.(*ast.Ident)

	return isIdent && ident.Name == contextPkgPath
}
