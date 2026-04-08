package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// chanDirectionAnalyzer returns an analyzer that flags bidirectional channel
// parameters in function signatures. Callers should receive the narrowest
// channel direction they need (<-chan or chan<-) so the compiler can enforce
// correct usage at call sites.
func chanDirectionAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "chandirection",
		Doc:      "flags bidirectional chan in function parameters; prefer <-chan or chan<- for direction safety",
		Run:      runChanDirection,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runChanDirection(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.FuncType)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		funcType, isFuncType := node.(*ast.FuncType)
		if !isFuncType {
			return
		}

		if funcType.Params == nil {
			return
		}

		for _, field := range funcType.Params.List {
			chanType, isChan := field.Type.(*ast.ChanType)
			if !isChan {
				continue
			}

			// ast.SEND | ast.RECV == 3, which means bidirectional.
			if chanType.Dir == ast.SEND|ast.RECV {
				pass.Reportf(field.Pos(), "bidirectional chan in parameter; use <-chan or chan<- for direction safety")
			}
		}
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
