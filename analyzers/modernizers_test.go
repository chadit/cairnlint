package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/passes/modernize"

	"github.com/chadit/cairnlint/analyzers"
)

// excludedModernizers maps each upstream modernize analyzer that cairnlint
// leaves unregistered to the reason it stays out.
//
// Deliberately a second copy of the list in modernizers.go rather than a
// reference to it. A test that reads the production list would agree with
// whatever that list says, including when it says nothing useful.
func excludedModernizers() map[string]string {
	return map[string]string{
		"errorsastype":   "prefererrorsastype covers it",
		"stringsbuilder": "stringconcatinloop covers it",
		"testingcontext": "contextbackground covers it",
		"newexpr":        "its facts force every dependency to load from source",
	}
}

// TestExcludedModernizersStillExistUpstream guards the exclusion list against
// a golang.org/x/tools bump that renames or drops an analyzer, which would
// register it again under the new name. That change compiles and passes every
// other test, so a self-merging dependency update would reach main unnoticed.
func TestExcludedModernizersStillExistUpstream(t *testing.T) {
	t.Parallel()

	upstream := make(map[string]bool, len(modernize.Suite))
	for _, analyzer := range modernize.Suite {
		upstream[analyzer.Name] = true
	}

	for name, reason := range excludedModernizers() {
		if !upstream[name] {
			t.Errorf("modernize.Suite no longer has %q: the exclusion in modernizers.go is dead, and it was there because %s", name, reason)
		}
	}
}

// TestExcludedModernizersAreNotRegistered verifies the exclusion takes effect,
// so no finding is reported by both an upstream analyzer and its native
// counterpart, and newexpr does not bring its facts back.
func TestExcludedModernizersAreNotRegistered(t *testing.T) {
	t.Parallel()

	excluded := excludedModernizers()

	for _, analyzer := range analyzers.All() {
		if reason, isExcluded := excluded[analyzer.Name]; isExcluded {
			t.Errorf("upstream analyzer %q is registered even though %s", analyzer.Name, reason)
		}
	}
}

// TestModernizersAreRegistered checks that the upstream suite is actually wired
// in, so an x/tools change that empties or renames Suite fails here rather than
// silently reducing cairnlint to its native rules.
func TestModernizersAreRegistered(t *testing.T) {
	t.Parallel()

	excluded := excludedModernizers()

	all := analyzers.All()

	registered := make(map[string]bool, len(all))
	for _, analyzer := range all {
		registered[analyzer.Name] = true
	}

	var found int

	for _, analyzer := range modernize.Suite {
		if _, isExcluded := excluded[analyzer.Name]; isExcluded {
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
