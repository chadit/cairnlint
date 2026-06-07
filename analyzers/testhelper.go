package analyzers

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// testEntryPrefixes are the function-name prefixes the testing package runs
// directly. Such functions are not helpers, so they are exempt.
var testEntryPrefixes = []string{testPrefix, benchmarkPrefix, fuzzPrefix, "Example"} //nolint:gochecknoglobals // package-internal lookup table, not mutable state

// testHelperMarkerAnalyzer returns an analyzer that flags a test helper taking
// *testing.T/B/F as its first parameter that does not call t.Helper() as its
// first statement. Marking helpers means failures report the caller's line, not
// a line buried inside the helper.
func testHelperMarkerAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "testhelper",
		Doc:      "flags test helpers taking *testing.T/B/F that do not call t.Helper() first",
		Run:      runTestHelperMarker,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runTestHelperMarker(pass *analysis.Pass) (any, error) {
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

		if !isTestFile(pass, funcDecl) || funcDecl.Recv != nil {
			return
		}

		if isTestEntryName(funcDecl.Name.Name) {
			return
		}

		if funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) == 0 {
			return
		}

		recvName, isTesting := testingParam(pass.TypesInfo, funcDecl.Type.Params.List[0])
		if !isTesting || recvName == "" || recvName == "_" {
			return
		}

		if funcDecl.Body == nil || len(funcDecl.Body.List) == 0 {
			return
		}

		if callsHelperFirst(funcDecl.Body, recvName) {
			return
		}

		pass.Reportf(funcDecl.Name.Pos(), "test helper %q does not call %s.Helper(); add it as the first statement", funcDecl.Name.Name, recvName)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isTestEntryName reports whether name is a function the testing package runs
// directly (Test, Benchmark, Fuzz, Example), including the bare prefix itself.
func isTestEntryName(name string) bool {
	for _, prefix := range testEntryPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

// callsHelperFirst reports whether the first statement of body is recvName.Helper().
func callsHelperFirst(body *ast.BlockStmt, recvName string) bool {
	exprStmt, isExpr := body.List[0].(*ast.ExprStmt)
	if !isExpr {
		return false
	}

	call, isCall := exprStmt.X.(*ast.CallExpr)
	if !isCall {
		return false
	}

	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != "Helper" {
		return false
	}

	ident, isIdent := sel.X.(*ast.Ident)

	return isIdent && ident.Name == recvName
}
