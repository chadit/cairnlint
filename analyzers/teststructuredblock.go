package analyzers

import (
	"go/ast"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// testStructuredBlockPattern matches the structured section headers that
// project rules forbid in doc comments for Test/Benchmark/Fuzz/Example
// functions. Each header starts at the beginning of a comment line.
var testStructuredBlockPattern = regexp.MustCompile(
	`(?m)^//\s*(Workflow|Test Environment|Expected Behavior|Purpose|Simulates)\b`,
)

// testStructuredBlockAnalyzer returns an analyzer that flags structured
// section headers in test function doc comments. Workflow, Test
// Environment, Expected Behavior, Purpose, and Simulates blocks duplicate
// what the test body already shows and inflate maintenance cost.
func testStructuredBlockAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "teststructuredblock",
		Doc:  "flags Workflow/Test Environment/Expected Behavior/Purpose/Simulates blocks in test doc comments",
		Run:  runTestStructuredBlock,
	}
}

func runTestStructuredBlock(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename
		if !strings.HasSuffix(filename, "_test.go") {
			continue
		}

		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Doc == nil {
				continue
			}

			if !isTestFrameworkFunc(funcDecl.Name.Name) {
				continue
			}

			for _, comment := range funcDecl.Doc.List {
				if testStructuredBlockPattern.MatchString(comment.Text) {
					pass.Reportf(comment.Pos(),
						"structured section header in test doc comment; keep test comments to 1-3 prose sentences")
				}
			}
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
