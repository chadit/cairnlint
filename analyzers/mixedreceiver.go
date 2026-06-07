package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// mixedReceiverAnalyzer returns an analyzer that flags a type whose methods mix
// value and pointer receivers. Go convention keeps a type's receivers uniform:
// once any method needs a pointer receiver, the rest should use one too.
func mixedReceiverAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "mixedreceiver",
		Doc:  "flags a type with both value and pointer receivers; use one receiver style consistently",
		Run:  runMixedReceiver,
	}
}

func runMixedReceiver(pass *analysis.Pass) (any, error) {
	// firstPtr records the receiver style first seen for each type so later
	// methods can be compared against it in source order.
	firstPtr := make(map[string]bool)
	seen := make(map[string]bool)

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			funcDecl, isFunc := decl.(*ast.FuncDecl)
			if !isFunc || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
				continue
			}

			typeName, _, isPtr, ok := receiverInfo(funcDecl.Recv.List[0])
			if !ok {
				continue
			}

			if !seen[typeName] {
				seen[typeName] = true
				firstPtr[typeName] = isPtr

				continue
			}

			if firstPtr[typeName] != isPtr {
				pass.Reportf(funcDecl.Recv.List[0].Pos(), "type %s has both value and pointer receivers; use one style consistently", typeName)
			}
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
