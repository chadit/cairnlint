package analyzers

import (
	"golang.org/x/tools/go/analysis"
)

// WrapWithoutFixes wraps each analyzer so no diagnostic carries a suggested
// fix. cairnlint reports; it never edits.
//
// This is enforced on the data rather than on the -fix flag because the flag
// belongs to the multichecker driver, which registers it whether cairnlint
// wants it or not. The driver applies fixes by walking the SuggestedFixes on
// the diagnostics it recorded through pass.Report, so clearing that field on
// the way through leaves it nothing to write no matter how it is invoked.
//
// Third-party analyzers make this necessary. cairnlint's own rules emit no
// fixes, but the modernize suite emits one for nearly every diagnostic, and
// those would otherwise become live edits the moment anyone passed -fix.
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
