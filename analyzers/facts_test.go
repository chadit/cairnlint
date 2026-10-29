package analyzers_test

import (
	"slices"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/modernize"

	"github.com/chadit/cairnlint/analyzers"
)

// factDeclarers walks roots and everything their Requires chains reach, and
// returns the name of each analyzer that declares a fact type. The walk is
// the one the multichecker does before it picks a load mode.
func factDeclarers(roots []*analysis.Analyzer) []string {
	var names []string

	seen := make(map[*analysis.Analyzer]bool)

	queue := slices.Clone(roots)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if seen[current] {
			continue
		}

		seen[current] = true

		if len(current.FactTypes) > 0 {
			names = append(names, current.Name)
		}

		queue = append(queue, current.Requires...)
	}

	return names
}

// TestNoRegisteredAnalyzerDeclaresFacts guards the load mode. One analyzer
// with facts anywhere in the Requires closure makes the multichecker load
// every dependency from source, the standard library included, and no
// cairnlint rule uses facts to pay for that.
func TestNoRegisteredAnalyzerDeclaresFacts(t *testing.T) {
	t.Parallel()

	registered := slices.Concat(analyzers.All(), analyzers.AgentOnly())

	if names := factDeclarers(registered); len(names) > 0 {
		t.Errorf("analyzers declaring facts: %v; each one switches the multichecker to packages.LoadAllSyntax", names)
	}
}

// TestFactWalkFindsUpstreamNewexpr is the control for the test above. It
// proves the walk detects a fact-bearing analyzer, and it fails once upstream
// newexpr stops declaring facts, which is the cue to register it again.
func TestFactWalkFindsUpstreamNewexpr(t *testing.T) {
	t.Parallel()

	if names := factDeclarers(modernize.Suite); !slices.Contains(names, "newexpr") {
		t.Errorf("fact walk over modernize.Suite found %v, want newexpr among them", names)
	}
}
