package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// synctestSleepWaitAnalyzer returns an analyzer that flags a time.Sleep call
// followed directly by synctest.Wait, which Go 1.27's synctest.Sleep combines.
// As two calls, a later edit can insert a statement between them or drop the
// Wait, changing what the test synchronizes on with no signal.
func synctestSleepWaitAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "synctestsleepwait",
		Doc:      "flags time.Sleep immediately followed by synctest.Wait; use synctest.Sleep instead (Go 1.27+)",
		Run:      runSynctestSleepWait,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runSynctestSleepWait(pass *analysis.Pass) (any, error) {
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

		checkBlockForSleepThenWait(block.List, pass)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkBlockForSleepThenWait scans adjacent statement pairs for the Sleep-then-
// Wait sequence. Adjacency is required because any statement between the two
// runs before the bubble settles, and synctest.Sleep would move it after.
func checkBlockForSleepThenWait(stmts []ast.Stmt, pass *analysis.Pass) {
	for idx := range len(stmts) - 1 {
		sleep := exprStmtCall(stmts[idx])
		if sleep == nil || !isCallTo(sleep, pass.TypesInfo, timePkgPath, "Sleep") {
			continue
		}

		wait := exprStmtCall(stmts[idx+1])
		if wait == nil || !isCallTo(wait, pass.TypesInfo, synctestPkgPath, "Wait") {
			continue
		}

		if !goVersionAtLeast(pass, sleep.Pos(), goVersion127) {
			continue
		}

		pass.Reportf(sleep.Pos(), "time.Sleep followed by synctest.Wait can be replaced with synctest.Sleep (Go 1.27)")
	}
}
