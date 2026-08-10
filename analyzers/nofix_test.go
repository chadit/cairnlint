package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chadit/cairnlint/analyzers"
)

// modernizerName is an upstream analyzer that ships a suggested fix, used to
// prove the stripping rather than assuming the analyzer had no fix to begin
// with.
const modernizerName = "any"

// TestUpstreamAnalyzerOffersFixUnwrapped is the control for
// TestAllStripsSuggestedFixes. Without it, a stripping test would still pass if
// upstream stopped emitting fixes, and cairnlint would lose the guarantee
// without any test noticing.
func TestUpstreamAnalyzerOffersFixUnwrapped(t *testing.T) {
	t.Parallel()

	bare := findCategorizedAnalyzer(t, modernizerName)

	results := analysistest.Run(t, analysistest.TestData(), bare, "nofix")

	if countSuggestedFixes(results) == 0 {
		t.Fatalf("analyzer %q offered no suggested fixes, so the stripping test proves nothing", modernizerName)
	}
}

// TestAllStripsSuggestedFixes verifies that no analyzer returned by All can
// carry an edit. Drivers apply fixes by walking SuggestedFixes on recorded
// diagnostics, so an empty field is what makes `-fix` and golangci-lint's
// `--fix` unable to rewrite source through cairnlint.
func TestAllStripsSuggestedFixes(t *testing.T) {
	t.Parallel()

	wrapped := findRegisteredAnalyzer(t, modernizerName)

	results := analysistest.Run(t, analysistest.TestData(), wrapped, "nofix")

	if got := countSuggestedFixes(results); got != 0 {
		t.Errorf("All() analyzer %q produced %d suggested fixes, want 0", modernizerName, got)
	}
}

// TestAgentOnlyStripsSuggestedFixes covers the second exported entry point, so
// enabling agent mode cannot reintroduce fixable diagnostics.
func TestAgentOnlyStripsSuggestedFixes(t *testing.T) {
	t.Parallel()

	for _, analyzer := range analyzers.AgentOnly() {
		if analyzer.Run == nil {
			t.Errorf("agent analyzer %q has no Run function", analyzer.Name)
		}
	}

	results := analysistest.Run(t, analysistest.TestData(), findAgentAnalyzer("agentstubbody"), "agentstubbody")

	if got := countSuggestedFixes(results); got != 0 {
		t.Errorf("AgentOnly() produced %d suggested fixes, want 0", got)
	}
}

// countSuggestedFixes totals the suggested fixes across every diagnostic in
// the results.
func countSuggestedFixes(results []*analysistest.Result) int {
	var total int

	for _, result := range results {
		for _, diag := range result.Diagnostics {
			total += len(diag.SuggestedFixes)
		}
	}

	return total
}

// findCategorizedAnalyzer returns the analyzer as registered in Categories,
// before All applies any wrapping.
func findCategorizedAnalyzer(t *testing.T, name string) *analysis.Analyzer {
	t.Helper()

	for _, cat := range analyzers.Categories() {
		for _, analyzer := range cat.Analyzers {
			if analyzer.Name == name {
				return analyzer
			}
		}
	}

	t.Fatalf("analyzer %q not found in Categories()", name)

	return nil
}

// findRegisteredAnalyzer returns the analyzer as All exposes it to drivers.
func findRegisteredAnalyzer(t *testing.T, name string) *analysis.Analyzer {
	t.Helper()

	for _, analyzer := range analyzers.All() {
		if analyzer.Name == name {
			return analyzer
		}
	}

	t.Fatalf("analyzer %q not found in All()", name)

	return nil
}
