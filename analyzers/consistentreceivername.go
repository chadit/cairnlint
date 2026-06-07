package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// consistentReceiverNameAnalyzer returns an analyzer that flags a type whose
// methods use different receiver variable names. Go convention uses the same
// short name on every method of a type so the code reads uniformly.
func consistentReceiverNameAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "consistentreceivername",
		Doc:  "flags inconsistent receiver names across a type's methods; pick one short name and reuse it",
		Run:  runConsistentReceiverName,
	}
}

func runConsistentReceiverName(pass *analysis.Pass) (any, error) {
	// firstName records the receiver name first seen for each type so later
	// methods can be compared against it in source order.
	firstName := make(map[string]string)

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			funcDecl, isFunc := decl.(*ast.FuncDecl)
			if !isFunc || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
				continue
			}

			typeName, recvName, _, ok := receiverInfo(funcDecl.Recv.List[0])
			if !ok || recvName == "" || recvName == "_" {
				continue
			}

			existing, found := firstName[typeName]
			if !found {
				firstName[typeName] = recvName

				continue
			}

			if existing != recvName {
				pass.Reportf(funcDecl.Recv.List[0].Names[0].Pos(), "receiver name %q is inconsistent with %q used on other methods of %s", recvName, existing, typeName)
			}
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
