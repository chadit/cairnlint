package analyzers

import (
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// suppressPattern matches nolint directives in comments with optional
// analyzer names (bare, single, or comma-separated).
var suppressPattern = regexp.MustCompile(`//\s*nolint(?::(\S+))?`)

// WrapWithNolint wraps each analyzer so diagnostics on lines containing
// a matching //nolint directive are suppressed. This gives standalone
// cairnlint the same suppression behavior as golangci-lint.
func WrapWithNolint(all []*analysis.Analyzer) []*analysis.Analyzer {
	wrapped := make([]*analysis.Analyzer, len(all))
	for idx, orig := range all {
		wrapped[idx] = wrapAnalyzer(orig)
	}

	return wrapped
}

func wrapAnalyzer(orig *analysis.Analyzer) *analysis.Analyzer {
	originalRun := orig.Run

	wrapped := *orig
	wrapped.Run = func(pass *analysis.Pass) (any, error) {
		nolintLines := buildNolintMap(pass, orig.Name)

		originalPass := *pass
		originalReport := pass.Report

		originalPass.Report = func(diag analysis.Diagnostic) {
			line := pass.Fset.Position(diag.Pos).Line
			if nolintLines[line] {
				return
			}

			originalReport(diag)
		}

		return originalRun(&originalPass)
	}

	return &wrapped
}

// buildNolintMap scans all comments in the pass and returns a set of
// line numbers where a //nolint directive suppresses the given analyzer.
func buildNolintMap(pass *analysis.Pass, analyzerName string) map[int]bool {
	result := make(map[int]bool)

	for _, file := range pass.Files {
		for _, cg := range file.Comments {
			for _, comment := range cg.List {
				if !strings.Contains(comment.Text, "nolint") {
					continue
				}

				match := suppressPattern.FindStringSubmatch(comment.Text)
				if match == nil {
					continue
				}

				line := pass.Fset.Position(comment.Pos()).Line

				// Bare //nolint suppresses all analyzers on this line.
				if match[1] == "" {
					result[line] = true

					continue
				}

				for name := range strings.SplitSeq(match[1], ",") {
					if strings.TrimSpace(name) == analyzerName {
						result[line] = true

						break
					}
				}
			}
		}
	}

	return result
}
