package main_test

import (
	"os"
	"testing"

	cairnlint "github.com/chadit/cairnlint"
)

// TestConsumeTagsFlag exercises every supported form for passing build
// tags (=value and space-separated, single and double dash) and confirms
// the flag is stripped from os.Args so multichecker does not see it.
//
//nolint:paralleltest // mutates os.Args, which is process-global; parallel would race
func TestConsumeTagsFlag(t *testing.T) {
	cases := []struct {
		name     string
		argv     []string
		wantTags string
		wantArgv []string
	}{
		{
			name:     "double dash equals",
			argv:     []string{"cairnlint", "--tags=integration", "./..."},
			wantTags: "integration",
			wantArgv: []string{"cairnlint", "./..."},
		},
		{
			name:     "single dash equals",
			argv:     []string{"cairnlint", "-tags=integration,e2e", "./..."},
			wantTags: "integration,e2e",
			wantArgv: []string{"cairnlint", "./..."},
		},
		{
			name:     "double dash space separated",
			argv:     []string{"cairnlint", "--tags", "integration", "./..."},
			wantTags: "integration",
			wantArgv: []string{"cairnlint", "./..."},
		},
		{
			name:     "single dash space separated",
			argv:     []string{"cairnlint", "-tags", "integration", "./..."},
			wantTags: "integration",
			wantArgv: []string{"cairnlint", "./..."},
		},
		{
			name:     "no tags flag",
			argv:     []string{"cairnlint", "./..."},
			wantTags: "",
			wantArgv: []string{"cairnlint", "./..."},
		},
		{
			name:     "last value wins when duplicated",
			argv:     []string{"cairnlint", "-tags=first", "-tags=second", "./..."},
			wantTags: "second",
			wantArgv: []string{"cairnlint", "./..."},
		},
		{
			name:     "space form at end without value is dropped",
			argv:     []string{"cairnlint", "-tags"},
			wantTags: "",
			wantArgv: []string{"cairnlint"},
		},
	}

	origArgs := os.Args

	t.Cleanup(func() {
		os.Args = origArgs
	})

	//nolint:paralleltest // subtests share os.Args with the parent; serial execution required
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			os.Args = append([]string{}, testCase.argv...)

			got := cairnlint.ConsumeTagsFlag()
			if got != testCase.wantTags {
				t.Errorf("tags: got %q, want %q", got, testCase.wantTags)
			}

			if !stringsEqual(os.Args, testCase.wantArgv) {
				t.Errorf("os.Args: got %v, want %v", os.Args, testCase.wantArgv)
			}
		})
	}
}

// TestPropagateBuildTagsPrepends confirms GOFLAGS is set when empty and
// prefixed (not overwritten) when it already has other values, so users
// who rely on GOFLAGS for other flags do not lose them.
func TestPropagateBuildTagsPrepends(t *testing.T) {
	cases := []struct {
		name        string
		initial     string
		tags        string
		wantGOFLAGS string
	}{
		{
			name:        "empty initial",
			initial:     "",
			tags:        "integration",
			wantGOFLAGS: "-tags=integration",
		},
		{
			name:        "preserves existing GOFLAGS",
			initial:     "-mod=vendor",
			tags:        "integration",
			wantGOFLAGS: "-tags=integration -mod=vendor",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("GOFLAGS", testCase.initial)

			if err := cairnlint.PropagateBuildTags(testCase.tags); err != nil {
				t.Fatalf("propagateBuildTags: %v", err)
			}

			got := os.Getenv("GOFLAGS")
			if got != testCase.wantGOFLAGS {
				t.Errorf("GOFLAGS: got %q, want %q", got, testCase.wantGOFLAGS)
			}
		})
	}
}

// stringsEqual reports whether two string slices have the same contents
// in the same order. Replaces slices.Equal to keep this file dependency-free.
func stringsEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	for idx, val := range left {
		if val != right[idx] {
			return false
		}
	}

	return true
}
