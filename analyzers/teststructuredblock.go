package analyzers

import (
	"go/ast"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// testStructuredBlockPattern matches any comment line starting with a listed
// label, bold or not, with a capitalized or lowercase second word. A wrapped
// prose line starting with a label is reported on purpose, matching dotai's
// c19 clause. Only spaces may follow "//", because a tab marks a gofmt code
// sample. Markdown headings and list items are not matched.
var testStructuredBlockPattern = regexp.MustCompile(
	`(?m)^// *\**(Workflow|Test [Ee]nvironment|Environment|Expected [Bb]ehavior|Purpose|Simulates)\b`,
)

// testStructuredBlockAnalyzer returns an analyzer that flags structured
// section headers in test function doc comments. Workflow, Environment,
// Expected Behavior, Purpose, and Simulates blocks duplicate what the test
// body already shows and inflate maintenance cost.
func testStructuredBlockAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "teststructuredblock",
		Doc:  "flags Workflow/Environment/Expected Behavior/Purpose/Simulates blocks in test doc comments, except on TestMain",
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

			// TestMain is package setup, not a test case, so the rule
			// clause leaves its doc comment alone.
			name := funcDecl.Name.Name
			if name == "TestMain" || !isTestFrameworkFunc(name) {
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
