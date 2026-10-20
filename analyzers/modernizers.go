package analyzers

import (
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/modernize"
)

// supersededModernizers are upstream analyzers cairnlint does not register,
// because a cairnlint rule already reports the same line with other wording:
//
//   - errorsastype: prefererrorsastype flags every errors.As call, a superset.
//   - stringsbuilder: stringconcatinloop covers the same += loop.
//   - testingcontext: contextbackground, contexttodo, and
//     wrappedcontextbackground already push tests toward t.Context().
var supersededModernizers = []string{ //nolint:gochecknoglobals // package-internal lookup table
	"errorsastype",
	"stringsbuilder",
	"testingcontext",
}

// modernizeAnalyzers returns the upstream modernize suite minus the analyzers
// cairnlint already covers. Taking the suite `go fix` runs keeps the rules
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
