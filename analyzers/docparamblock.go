package analyzers

import (
	"go/ast"
	"regexp"

	"golang.org/x/tools/go/analysis"
)

// docParamBlockPattern matches Javadoc-style Parameters: and Returns:
// section headers in Go doc comments. Anchoring to the start of the
// comment line keeps prose mentions like "the parameters we accept..."
// from matching; the trailing colon is what distinguishes a section
// header from ordinary prose.
var docParamBlockPattern = regexp.MustCompile(`^//\s*(Parameters|Returns)\s*:`)

// docParamBlockAnalyzer returns an analyzer that flags Javadoc-style
// `Parameters:` / `Returns:` headers in non-test function doc comments.
// Go convention weaves parameter and return information into prose, and
// godoc renders these headers as plain text without extra structure.
func docParamBlockAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "docparamblock",
		Doc:  "flags Javadoc-style Parameters: / Returns: blocks in non-test function doc comments",
		Run:  runDocParamBlock,
	}
}

func runDocParamBlock(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Doc == nil {
				continue
			}

			if isTestFrameworkFunc(funcDecl.Name.Name) {
				continue
			}

			for _, comment := range funcDecl.Doc.List {
				if docParamBlockPattern.MatchString(comment.Text) {
					pass.Reportf(comment.Pos(),
						"Javadoc-style section header in doc comment; weave parameters and returns into prose")
				}
			}
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
