package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// contextFirstParamAnalyzer returns an analyzer that flags a context.Context
// parameter that is not first in the signature. Go convention puts the context
// first so call sites read consistently. Functions whose first parameter is a
// *testing.T/B/F are skipped, since the testing value leads in test helpers.
func contextFirstParamAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "contextfirstparam",
		Doc:      "flags context.Context that is not the first parameter; pass it first",
		Run:      runContextFirstParam,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runContextFirstParam(pass *analysis.Pass) (any, error) {
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

		checkContextFirst(pass, funcDecl.Type)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkContextFirst reports a context.Context parameter that appears after the
// first flattened parameter position.
func checkContextFirst(pass *analysis.Pass, funcType *ast.FuncType) {
	if funcType.Params == nil || len(funcType.Params.List) == 0 {
		return
	}

	// A leading testing value (test helpers) takes precedence over the context.
	if _, isTesting := testingParam(pass.TypesInfo, funcType.Params.List[0]); isTesting {
		return
	}

	var position int

	for _, field := range funcType.Params.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}

		if isContextType(field.Type) && position != 0 {
			pass.Reportf(field.Pos(), "context.Context should be the first parameter")

			return
		}

		position += count
	}
}
