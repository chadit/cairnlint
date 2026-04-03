package analyzers

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// noForTestFuncAnalyzer returns an analyzer that flags functions whose names
// end with ForTest or ForTesting. These functions export internals for test
// use. Tests should exercise the public API instead.
func noForTestFuncAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "nofortestfunc",
		Doc:  "flags functions with ForTest/ForTesting suffix; test through the public API instead",
		Run:  runNoForTestFunc,
	}
}

func runNoForTestFunc(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			funcDecl, isFuncDecl := node.(*ast.FuncDecl)
			if !isFuncDecl {
				return true
			}

			name := funcDecl.Name.Name
			if strings.HasSuffix(name, "ForTest") || strings.HasSuffix(name, "ForTesting") {
				pass.Reportf(funcDecl.Name.Pos(), "function %s exports internals for testing; test through the public API", name)
			}

			return true
		})
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
