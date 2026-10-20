package analyzers

import (
	"golang.org/x/tools/go/analysis"
)

// WrapWithoutFixes wraps each analyzer so no diagnostic carries a suggested
// fix. cairnlint reports; it never edits. The multichecker driver registers
// -fix regardless and applies the SuggestedFixes recorded through pass.Report,
// so clearing that field holds however the driver is invoked. The modernize
// suite needs it, since it attaches a fix to nearly every diagnostic.
func WrapWithoutFixes(all []*analysis.Analyzer) []*analysis.Analyzer {
	wrapped := make([]*analysis.Analyzer, len(all))
	for idx, orig := range all {
		wrapped[idx] = stripFixes(orig)
	}

	return wrapped
}

func stripFixes(orig *analysis.Analyzer) *analysis.Analyzer {
	originalRun := orig.Run

	wrapped := *orig
	wrapped.Run = func(pass *analysis.Pass) (any, error) {
		strippedPass := *pass
		originalReport := pass.Report

		strippedPass.Report = func(diag analysis.Diagnostic) {
			diag.SuggestedFixes = nil

			originalReport(diag)
		}

		return originalRun(&strippedPass)
	}

	return &wrapped
}
