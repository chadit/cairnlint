package analyzers

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// noInlineMocksAnalyzer returns an analyzer that flags `type MockFoo struct`
// declarations in test files outside test/mocks/. Mock types should be
// centralized in test/mocks/ so they're discoverable and reusable.
func noInlineMocksAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "noinlinemocks",
		Doc:  "flags mock struct types in test files outside test/mocks/; centralize mocks for reuse",
		Run:  runNoInlineMocks,
	}
}

func runNoInlineMocks(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename

		if !strings.HasSuffix(filename, "_test.go") {
			continue
		}

		if strings.Contains(filename, "test/mocks/") {
			continue
		}

		checkFileForInlineMocks(pass, file)
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkFileForInlineMocks walks type declarations in a test file and flags
// struct types whose name starts with Mock or mock.
func checkFileForInlineMocks(pass *analysis.Pass, file *ast.File) {
	ast.Inspect(file, func(node ast.Node) bool {
		typeSpec, isTypeSpec := node.(*ast.TypeSpec)
		if !isTypeSpec {
			return true
		}

		_, isStruct := typeSpec.Type.(*ast.StructType)
		if !isStruct {
			return true
		}

		name := typeSpec.Name.Name
		if strings.HasPrefix(name, "Mock") || strings.HasPrefix(name, "mock") {
			pass.Reportf(typeSpec.Name.Pos(), "mock type %s must be in test/mocks/, not inline in test files", name)
		}

		return true
	})
}
