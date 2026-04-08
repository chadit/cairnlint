package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// reflectInLoopAnalyzer returns an analyzer that flags reflect.ValueOf and
// reflect.TypeOf calls inside loop bodies. Reflection allocates on every call,
// so hoisting the result outside the loop avoids repeated work.
func reflectInLoopAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "reflectinloop",
		Doc:      "flags reflect.ValueOf/TypeOf inside loops; reflection is expensive per-call, hoist outside or use an interface",
		Run:      runReflectInLoop,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runReflectInLoop(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.WithStack(nodeFilter, func(node ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}

		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel {
			return true
		}

		funcName := sel.Sel.Name
		if funcName != "ValueOf" && funcName != "TypeOf" {
			return true
		}

		if !isCallTo(call, pass.TypesInfo, "reflect", funcName) {
			return true
		}

		if !isInsideLoop(stack) {
			return true
		}

		pass.Reportf(
			call.Pos(),
			"reflect.%s() inside loop; reflection is expensive, hoist outside or use an interface",
			funcName,
		)

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
