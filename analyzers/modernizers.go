package analyzers

import (
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/modernize"
)

// supersededModernizers are upstream analyzers cairnlint does not register.
// The first three duplicate a cairnlint rule that reports the same line with
// other wording. newexpr is no duplicate; it is left out for what it costs.
var supersededModernizers = []string{ //nolint:gochecknoglobals // package-internal lookup table
	// prefererrorsastype flags every errors.As call, a superset.
	"errorsastype",
	// stringconcatinloop covers the same += loop.
	"stringsbuilder",
	// contextbackground, contexttodo, and wrappedcontextbackground already
	// push tests toward t.Context().
	"testingcontext",
	// newexpr declares FactTypes. One analyzer with facts switches the
	// multichecker to packages.LoadAllSyntax, which type-checks every
	// dependency from source, the standard library included, on every run.
	// No cairnlint rule needs facts, so one modernizer should not set that
	// cost for all of them.
	"newexpr",
}

// modernizeAnalyzers returns the upstream modernize suite minus
// supersededModernizers. Taking the suite `go fix` runs keeps the rules
// current without tracking each Go release by hand, and [WrapWithoutFixes]
// strips their fixes. Upstream leaves bloop and fmtappendf out of Suite, so
// cairnlint keeps [preferBLoopAnalyzer] and [preferFmtAppendfAnalyzer].
func modernizeAnalyzers() []*analysis.Analyzer {
	out := make([]*analysis.Analyzer, 0, len(modernize.Suite))

	for _, analyzer := range modernize.Suite {
		if slices.Contains(supersededModernizers, analyzer.Name) {
			continue
		}

		out = append(out, analyzer)
	}

	return out
}
