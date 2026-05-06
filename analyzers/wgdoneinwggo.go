package analyzers

import (
	"go/ast"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// wgDoneInWGGoAnalyzer returns an analyzer that flags wg.Done() called inside
// a wg.Go() closure on the same WaitGroup. WaitGroup.Go (Go 1.25) already
// calls Done when f returns, so an explicit Done double-decrements the counter
// and causes panic or early Wait return.
func wgDoneInWGGoAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "wgdoneinwggo",
		Doc:      "flags wg.Done inside wg.Go closure; WaitGroup.Go calls Done automatically when f returns, explicit Done double-decrements",
		Run:      runWGDoneInWGGo,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runWGDoneInWGGo(pass *analysis.Pass) (any, error) {
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

		// Check if this is a .Done() call on a WaitGroup.
		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel || sel.Sel.Name != "Done" {
			return true
		}

		if !isWaitGroupReceiver(sel.X, pass.TypesInfo) && !isWaitGroupMethod(sel, pass.TypesInfo) {
			return true
		}

		doneReceiver := receiverIdent(sel.X)
		if doneReceiver == "" {
			return true
		}

		// Walk the stack backwards looking for a FuncLit whose parent is
		// a wg.Go() call on the same WaitGroup receiver.
		if isInsideWGGoClosure(stack, doneReceiver, pass) {
			pass.Reportf(call.Pos(), "wg.Done inside wg.Go is redundant; WaitGroup.Go calls Done automatically when f returns")
		}

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isInsideWGGoClosure walks the stack backwards looking for a function literal
// that is passed as an argument to wg.Go() on the same receiver as the Done
// call. Returns true when the Done is nested inside such a closure.
func isInsideWGGoClosure(stack []ast.Node, doneReceiver string, pass *analysis.Pass) bool {
	for idx := range slices.Backward(stack) {
		funcLit, isFuncLit := stack[idx].(*ast.FuncLit)
		if !isFuncLit {
			continue
		}

		if idx == 0 {
			return false
		}

		parentCall, isCall := stack[idx-1].(*ast.CallExpr)
		if !isCall {
			continue
		}

		parentSel, isSel := parentCall.Fun.(*ast.SelectorExpr)
		if !isSel || parentSel.Sel.Name != goMethodName {
			continue
		}

		if !isWaitGroupReceiver(parentSel.X, pass.TypesInfo) && !isWaitGroupMethod(parentSel, pass.TypesInfo) {
			continue
		}

		goReceiver := receiverIdent(parentSel.X)
		if goReceiver == "" {
			continue
		}

		// Only flag when Done and Go target the same WaitGroup and
		// the FuncLit is actually an argument to the Go call.
		if goReceiver == doneReceiver && isFuncLitArg(parentCall, funcLit) {
			return true
		}
	}

	return false
}
