package analyzers

import (
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/modernize"
)

// supersededModernizers are upstream analyzers cairnlint deliberately does not
// register, because a cairnlint rule already covers the same ground and would
// report the same line twice with different wording.
//
//   - errorsastype: cairnlint's prefererrorsastype flags every errors.As call.
//     Upstream fires only where the var declaration has the exact rewritable
//     shape, so it is the narrower of the two.
//   - stringsbuilder: cairnlint's stringconcatinloop covers the same += loop.
//   - testingcontext: cairnlint's contextbackground, contexttodo, and
//     wrappedcontextbackground already push tests toward t.Context().
var supersededModernizers = []string{ //nolint:gochecknoglobals // package-internal lookup table
	"errorsastype",
	"stringsbuilder",
	"testingcontext",
}

// modernizeAnalyzers returns the upstream modernize suite minus the analyzers
// cairnlint already covers.
//
// These come from golang.org/x/tools, the same suite `go fix` runs, so the
// rules stay current without cairnlint tracking each Go release by hand. Their
// suggested fixes are stripped before any diagnostic reaches the driver; see
// [WrapWithoutFixes].
//
// Two upstream analyzers are absent from Suite by upstream's own choice, and
// cairnlint keeps its own versions of both: bloop (see [preferBLoopAnalyzer])
// and fmtappendf (see [preferFmtAppendfAnalyzer]).
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
