package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// disallowedReceiverNames holds object-oriented receiver names that Go style
// avoids. Go receivers read like an abbreviation of the type, not a generic
// self-reference.
var disallowedReceiverNames = map[string]bool{ //nolint:gochecknoglobals // package-internal lookup table, not mutable state
	"this": true,
	"self": true,
	"me":   true,
}

// selfReceiverAnalyzer returns an analyzer that flags method receivers named
// this, self, or me. Go convention uses a short name derived from the type
// (one or two letters), not an object-oriented self-reference.
func selfReceiverAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "selfreceiver",
		Doc:      "flags receivers named this, self, or me; use a short name derived from the type",
		Run:      runSelfReceiver,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runSelfReceiver(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		funcDecl, isFuncDecl := node.(*ast.FuncDecl)
		if !isFuncDecl {
			return
		}

		if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			return
		}

		field := funcDecl.Recv.List[0]
		if len(field.Names) == 0 {
			return
		}

		name := field.Names[0].Name
		if disallowedReceiverNames[name] {
			pass.Reportf(field.Names[0].Pos(), "receiver name %q; use a short name derived from the type instead of this/self/me", name)
		}
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
