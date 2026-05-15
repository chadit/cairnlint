package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// wgDoneMethod is the method name on sync.WaitGroup that decrements the
// counter. Extracted to a constant so goconst does not flip over its
// default threshold as more analyzers reference WaitGroup internals.
const wgDoneMethod = "Done"

// preferWGGoAnalyzer returns an analyzer that flags the pre-Go-1.25 pattern
// `wg.Add(1); go func() { defer wg.Done(); ... }()` and suggests replacing
// it with `wg.Go(fn)`. WaitGroup.Go (Go 1.25) calls Add(1) and defers Done
// internally, so callers can drop both the manual Add and the deferred Done.
func preferWGGoAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "preferwggo",
		Doc:      "flags wg.Add(1) followed by go func(){ defer wg.Done(); ... }(); prefer wg.Go(fn) (Go 1.25)",
		Run:      runPreferWGGo,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runPreferWGGo(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.BlockStmt)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		block, isBlock := node.(*ast.BlockStmt)
		if !isBlock {
			return
		}

		checkBlockForPreferWGGo(block.List, pass)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkBlockForPreferWGGo scans adjacent statement pairs in a block for the
// classic pre-Go-1.25 WaitGroup pattern. Adjacency keeps the rewrite safe;
// any intervening statement could depend on timing between Add and the
// goroutine spawn, so we do not look past the next statement.
func checkBlockForPreferWGGo(stmts []ast.Stmt, pass *analysis.Pass) {
	for idx := range len(stmts) - 1 {
		addCall, addReceiver := extractWGAddOneCall(stmts[idx], pass.TypesInfo)
		if addCall == nil {
			continue
		}

		if !isMatchingGoWithDeferredDone(stmts[idx+1], addReceiver, pass.TypesInfo) {
			continue
		}

		pass.Reportf(addCall.Pos(), "wg.Add(1) + go func(){ defer wg.Done(); ... }() can be replaced with wg.Go(fn) (Go 1.25)")
	}
}

// extractWGAddOneCall returns the Add call when stmt is `wg.Add(1)` on a
// sync.WaitGroup, plus the receiver identifier. The argument must be the
// untyped literal 1 because wg.Go always increments the counter by exactly
// one; other values would silently change semantics if blindly migrated.
func extractWGAddOneCall(stmt ast.Stmt, info *types.Info) (*ast.CallExpr, string) {
	call, recv := extractWGAddCall(stmt, info)
	if call == nil {
		return nil, ""
	}

	if len(call.Args) != 1 {
		return nil, ""
	}

	lit, isLit := call.Args[0].(*ast.BasicLit)
	if !isLit || lit.Value != "1" {
		return nil, ""
	}

	return call, recv
}

// isMatchingGoWithDeferredDone reports whether stmt is a `go funcLit(...)`
// whose body starts with `defer wg.Done()` on the supplied receiver. The
// defer must be the first statement in the body so the Done fires on every
// exit path, matching wg.Go's guarantee.
func isMatchingGoWithDeferredDone(stmt ast.Stmt, receiver string, info *types.Info) bool {
	goStmt, isGo := stmt.(*ast.GoStmt)
	if !isGo {
		return false
	}

	funcLit, isFuncLit := goStmt.Call.Fun.(*ast.FuncLit)
	if !isFuncLit || funcLit.Body == nil || len(funcLit.Body.List) == 0 {
		return false
	}

	deferStmt, isDefer := funcLit.Body.List[0].(*ast.DeferStmt)
	if !isDefer {
		return false
	}

	sel, isSel := deferStmt.Call.Fun.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != wgDoneMethod {
		return false
	}

	if !isWaitGroupReceiver(sel.X, info) && !isWaitGroupMethod(sel, info) {
		return false
	}

	if len(deferStmt.Call.Args) != 0 {
		return false
	}

	return receiverIdent(sel.X) == receiver
}
