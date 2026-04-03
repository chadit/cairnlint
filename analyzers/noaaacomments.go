package analyzers

import (
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// aaaCommentPattern matches AAA section marker comments: // Arrange,
// // Act, or // Assert standing alone on a line.
var aaaCommentPattern = regexp.MustCompile(`^\s*//\s*(Arrange|Act|Assert)\s*$`)

// noAAACommentsAnalyzer returns an analyzer that flags // Arrange, // Act,
// and // Assert comments in test files. These AAA section markers add noise
// without helping readability; well-structured tests are self-explanatory.
func noAAACommentsAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "noaaacomments",
		Doc:  "flags // Arrange / // Act / // Assert comments in test files; these markers are not allowed",
		Run:  runNoAAAComments,
	}
}

func runNoAAAComments(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename

		if !strings.HasSuffix(filename, "_test.go") {
			continue
		}

		for _, cg := range file.Comments {
			for _, comment := range cg.List {
				if aaaCommentPattern.MatchString(comment.Text) {
					pass.Reportf(comment.Pos(), "AAA section marker not allowed; structure tests clearly without // Arrange / // Act / // Assert")
				}
			}
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
