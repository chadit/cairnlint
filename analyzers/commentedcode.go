package analyzers

import (
	"go/ast"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// commentedCodePattern matches comment text that looks like disabled Go code.
// Each alternative starts with a Go keyword that typically begins a statement.
var commentedCodePattern = regexp.MustCompile(
	`^\s*//\s*(func |if |for |return |var |defer |go |select \{|switch )`,
)

// commentedOutCodeAnalyzer returns an analyzer that flags comments in non-test
// Go files that look like disabled code. Commented-out code should be removed
// because git tracks history. Build tags or feature flags are the correct way
// to conditionally disable code.
func commentedOutCodeAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "commentedcode",
		Doc:  "flags comments that look like commented-out Go code; remove dead code or use build tags",
		Run:  runCommentedCode,
	}
}

func runCommentedCode(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		pos := pass.Fset.Position(file.Pos())
		if strings.HasSuffix(pos.Filename, "_test.go") {
			continue
		}

		for _, cg := range file.Comments {
			for _, comment := range cg.List {
				// Only flag line comments, not doc comments on declarations.
				if isDocComment(file, cg) {
					continue
				}

				if commentedCodePattern.MatchString(comment.Text) {
					pass.Reportf(comment.Pos(), "commented-out code; remove or use build tags")
				}
			}
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isDocComment reports whether cg is the doc comment for a top-level
// declaration in the given file. Doc comments explain the API and should
// not be flagged even if they happen to contain Go-like syntax.
func isDocComment(file *ast.File, commentGroup *ast.CommentGroup) bool {
	for _, decl := range file.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			if typed.Doc == commentGroup {
				return true
			}
		case *ast.FuncDecl:
			if typed.Doc == commentGroup {
				return true
			}
		}
	}

	return false
}
