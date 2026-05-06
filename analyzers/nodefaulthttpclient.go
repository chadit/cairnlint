package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// noDefaultHTTPClientAnalyzer returns an analyzer that flags references to
// http.DefaultClient. The default client has no timeout configured, so a
// hanging server blocks the calling goroutine indefinitely. Construct an
// http.Client with an explicit Timeout instead.
//
// Top-level http.Get/Post/Head/PostForm are already caught by noctx
// (missing context), so this analyzer focuses on direct DefaultClient use.
func noDefaultHTTPClientAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "nodefaulthttpclient",
		Doc:      "flags http.DefaultClient usage; construct an http.Client with explicit Timeout instead",
		Run:      runNoDefaultHTTPClient,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runNoDefaultHTTPClient(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.SelectorExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		sel, isSel := node.(*ast.SelectorExpr)
		if !isSel || sel.Sel.Name != "DefaultClient" {
			return
		}

		obj := pass.TypesInfo.ObjectOf(sel.Sel)
		if obj == nil {
			return
		}

		pkg := obj.Pkg()
		if pkg == nil || pkg.Path() != httpPkgPath {
			return
		}

		// Verify it's the package-level variable, not a field named DefaultClient
		if _, isVar := obj.(*types.Var); !isVar {
			return
		}

		pass.Reportf(sel.Pos(), "http.DefaultClient has no timeout; construct an http.Client with explicit Timeout")
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
