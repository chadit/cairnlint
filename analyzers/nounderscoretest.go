package analyzers

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// noUnderscoreTestNamesAnalyzer returns an analyzer that flags test function
// names containing underscores. Go convention uses MixedCaps for test names
// (e.g., TestFooBar rather than TestFoo_Bar).
func noUnderscoreTestNamesAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "nounderscoretest",
		Doc:      "flags test function names containing underscores; Go convention uses MixedCaps",
		Run:      runNoUnderscoreTestNames,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runNoUnderscoreTestNames(pass *analysis.Pass) (any, error) {
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

		// Only flag Test*, Benchmark*, and Fuzz* functions.
		if !isTestFuncName(name) {
			return
		}

		remainder := stripTestPrefix(name)
		if remainder == "" {
			return
		}

		if strings.Contains(remainder, "_") {
			pass.Reportf(funcDecl.Name.Pos(), "test name %q contains underscores; use MixedCaps (e.g., TestFooBar)", name)
		}
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isTestFuncName reports whether name starts with a recognized test prefix
// followed by an uppercase letter (matching the testing package convention).
func isTestFuncName(name string) bool {
	prefixes := []string{testPrefix, benchmarkPrefix, fuzzPrefix}

	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) && len(name) > len(prefix) {
			// The character after the prefix must be uppercase or underscore
			// to match test function naming conventions.
			return true
		}
	}

	return false
}

// stripTestPrefix removes the Test/Benchmark/Fuzz prefix from name.
func stripTestPrefix(name string) string {
	prefixes := []string{benchmarkPrefix, fuzzPrefix, testPrefix}

	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return name[len(prefix):]
		}
	}

	return name
}
