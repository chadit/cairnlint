package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// benchReportAllocsAnalyzer returns an analyzer that flags benchmark
// functions missing b.ReportAllocs(). Without it, allocation regressions
// are invisible in benchmark output.
func benchReportAllocsAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "benchreportallocs",
		Doc:      "flags Benchmark* functions without b.ReportAllocs(); allocation regressions are invisible without it",
		Run:      runBenchReportAllocs,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runBenchReportAllocs(pass *analysis.Pass) (any, error) {
	if !hasTestFiles(pass) {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

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

		if !isTestFile(pass, funcDecl) {
			return
		}

		if !isBenchmarkFunc(funcDecl, pass.TypesInfo) {
			return
		}

		paramName := benchParamName(funcDecl)
		if paramName == "" {
			return
		}

		if hasReportAllocsCall(funcDecl.Body, paramName) {
			return
		}

		pass.Reportf(funcDecl.Pos(), "benchmark missing b.ReportAllocs(); add it to track memory allocations")
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// benchParamName returns the name of the first parameter in a benchmark
// function declaration. Returns "" if the parameter is unnamed.
func benchParamName(funcDecl *ast.FuncDecl) string {
	params := funcDecl.Type.Params
	if params == nil || len(params.List) == 0 {
		return ""
	}

	firstParam := params.List[0]
	if len(firstParam.Names) == 0 {
		return ""
	}

	return firstParam.Names[0].Name
}

// hasReportAllocsCall walks the function body looking for a call to
// <paramName>.ReportAllocs().
func hasReportAllocsCall(body *ast.BlockStmt, paramName string) bool {
	if body == nil {
		return false
	}

	var found bool

	ast.Inspect(body, func(node ast.Node) bool {
		if found {
			return false
		}

		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}

		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel || sel.Sel.Name != "ReportAllocs" {
			return true
		}

		ident, isIdent := sel.X.(*ast.Ident)
		if !isIdent || ident.Name != paramName {
			return true
		}

		found = true

		return false
	})

	return found
}
