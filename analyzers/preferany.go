package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// preferAnyAnalyzer returns an analyzer that flags the empty interface type
// written as interface{}. Since Go 1.18 the predeclared any alias reads more
// clearly and is the convention in new code.
func preferAnyAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "preferany",
		Doc:      "flags the empty interface{} type; use the any alias instead (Go 1.18+)",
		Run:      runPreferAny,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runPreferAny(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.InterfaceType)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		iface, isIface := node.(*ast.InterfaceType)
		if !isIface {
			return
		}

		// A non-empty method or constraint list means this is a real interface,
		// not the empty interface{} that any aliases.
		if iface.Methods != nil && len(iface.Methods.List) > 0 {
			return
		}

		pass.Reportf(iface.Pos(), "use any instead of interface{}")
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
