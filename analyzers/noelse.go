package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// noElseAnalyzer returns an analyzer that flags if-else blocks. Idiomatic Go
// prefers early returns and guard clauses, which reduce nesting and make the
// happy path obvious.
func noElseAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "noelse",
		Doc:      "flags if-else blocks; prefer early returns and guard clauses",
		Run:      runNoElse,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runNoElse(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.IfStmt)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		ifStmt, isIf := node.(*ast.IfStmt)
		if !isIf {
			return
		}

		if ifStmt.Else == nil {
			return
		}

		pass.Reportf(ifStmt.Else.Pos(), "rewrite if-else as early return or guard clause")
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
