package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/passes/modernize"

	"github.com/chadit/cairnlint/analyzers"
)

// supersededByCairnlint maps each upstream modernize analyzer that cairnlint
// leaves unregistered to the native rule covering the same ground.
//
// Deliberately a second copy of the list in modernizers.go rather than a
// reference to it. A test that reads the production list would agree with
// whatever that list says, including when it says nothing useful.
func supersededByCairnlint() map[string]string {
	return map[string]string{
		"errorsastype":   "prefererrorsastype",
		"stringsbuilder": "stringconcatinloop",
		"testingcontext": "contextbackground",
	}
}

// TestSupersededModernizersStillExistUpstream guards the exclusion list against
// a golang.org/x/tools bump.
//
// The list matches upstream analyzers by name. If upstream renames or drops
// one, the exclusion stops matching, the analyzer quietly registers again, and
// every finding it shares with the native rule gets reported twice. That change
// compiles and passes every other test, so with dependency updates merging
// themselves it would otherwise reach main unnoticed.
func TestSupersededModernizersStillExistUpstream(t *testing.T) {
	t.Parallel()

	upstream := make(map[string]bool, len(modernize.Suite))
	for _, analyzer := range modernize.Suite {
		upstream[analyzer.Name] = true
	}

	for name, native := range supersededByCairnlint() {
		if !upstream[name] {
			t.Errorf("modernize.Suite no longer has %q: the exclusion in modernizers.go is dead, so %q now has a duplicate", name, native)
		}
	}
}

// TestSupersededModernizersAreNotRegistered verifies the exclusion takes
// effect, so no finding is reported by both an upstream analyzer and its
// native counterpart.
func TestSupersededModernizersAreNotRegistered(t *testing.T) {
	t.Parallel()

	superseded := supersededByCairnlint()

	for _, analyzer := range analyzers.All() {
		if native, covered := superseded[analyzer.Name]; covered {
			t.Errorf("upstream analyzer %q is registered even though %q already covers it", analyzer.Name, native)
		}
	}
}

// TestModernizersAreRegistered checks that the upstream suite is actually wired
// in, so an x/tools change that empties or renames Suite fails here rather than
// silently reducing cairnlint to its native rules.
func TestModernizersAreRegistered(t *testing.T) {
	t.Parallel()

	superseded := supersededByCairnlint()

	registered := make(map[string]bool)
	for _, analyzer := range analyzers.All() {
		registered[analyzer.Name] = true
	}

	var found int

	for _, analyzer := range modernize.Suite {
		if _, covered := superseded[analyzer.Name]; covered {
			continue
		}

		if !registered[analyzer.Name] {
			t.Errorf("modernize analyzer %q is missing from All()", analyzer.Name)

			continue
		}

		found++
	}

	if found == 0 {
		t.Fatal("no modernize analyzers registered; the upstream suite is not wired in")
	}
}
