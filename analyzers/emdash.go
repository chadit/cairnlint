package analyzers

import (
	"strings"

	"golang.org/x/tools/go/analysis"
)

// emDashChar is the Unicode em dash (U+2014). Writing style rules forbid
// it in text: replace with a comma for a continuing thought or a period
// for a separate sentence.
const emDashChar = "—"

// emdashAnalyzer returns an analyzer that flags em dashes in comment text.
// Em dashes are a recognized AI-writing tell and project style rules
// require them to be replaced with a comma or a period.
func emdashAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "emdash",
		Doc:  "flags em dash (U+2014) in comments; use a comma or a period instead",
		Run:  runEmdash,
	}
}

func runEmdash(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		for _, cg := range file.Comments {
			for _, comment := range cg.List {
				if strings.Contains(comment.Text, emDashChar) {
					pass.Reportf(comment.Pos(), "em dash in comment; replace with a comma or a period")
				}
			}
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
