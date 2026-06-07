package analyzers

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// testUnderscorePrefixPrefix is the "Test_" token that signals a missing
// subject between the Test prefix and the first underscore.
const testUnderscorePrefixPrefix = testPrefix + "_"

// testUnderscorePrefixAnalyzer returns an analyzer that flags test function
// names with an underscore immediately after the Test prefix (e.g. Test_Foo).
// Go convention allows an underscore to separate the subject under test from a
// scenario (TestFoo_Bar), but the subject must not be empty. Benchmark and
// Fuzz names are left alone so their own grouping conventions still apply.
func testUnderscorePrefixAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "testunderscoreprefix",
		Doc:      "flags Test names with an underscore right after the prefix (Test_Foo); name the subject first (TestFoo_Bar)",
		Run:      runTestUnderscorePrefix,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runTestUnderscorePrefix(pass *analysis.Pass) (any, error) {
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

		name := funcDecl.Name.Name

		// Only Test names are checked. Benchmark_/Fuzz_ grouping is allowed.
		if !strings.HasPrefix(name, testUnderscorePrefixPrefix) {
			return
		}

		pass.Reportf(funcDecl.Name.Pos(), "test name %q has an underscore immediately after the Test prefix; name the subject under test before the underscore (e.g., TestSubject_scenario)", name)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
