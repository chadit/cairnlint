package main_test

import (
	"os"
	"testing"

	cairnlint "github.com/chadit/cairnlint"
)

const (
	testIntegrationTag  = "integration"
	testWildcardPattern = "./..."
	testBinaryName      = "cairnlint"
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
			argv:     []string{testBinaryName, "--tags=" + testIntegrationTag, testWildcardPattern},
			wantTags: testIntegrationTag,
			wantArgv: []string{testBinaryName, testWildcardPattern},
		},
		{
			name:     "single dash equals",
			argv:     []string{testBinaryName, "-tags=integration,e2e", testWildcardPattern},
			wantTags: "integration,e2e",
			wantArgv: []string{testBinaryName, testWildcardPattern},
		},
		{
			name:     "double dash space separated",
			argv:     []string{testBinaryName, "--tags", testIntegrationTag, testWildcardPattern},
			wantTags: testIntegrationTag,
			wantArgv: []string{testBinaryName, testWildcardPattern},
		},
		{
			name:     "single dash space separated",
			argv:     []string{testBinaryName, "-tags", testIntegrationTag, testWildcardPattern},
			wantTags: testIntegrationTag,
			wantArgv: []string{testBinaryName, testWildcardPattern},
		},
		{
			name:     "no tags flag",
			argv:     []string{testBinaryName, testWildcardPattern},
			wantTags: "",
			wantArgv: []string{testBinaryName, testWildcardPattern},
		},
		{
			name:     "last value wins when duplicated",
			argv:     []string{testBinaryName, "-tags=first", "-tags=second", testWildcardPattern},
			wantTags: "second",
			wantArgv: []string{testBinaryName, testWildcardPattern},
		},
		{
			name:     "space form at end without value is dropped",
			argv:     []string{testBinaryName, "-tags"},
			wantTags: "",
			wantArgv: []string{testBinaryName},
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
			tags:        testIntegrationTag,
			wantGOFLAGS: "-tags=" + testIntegrationTag,
		},
		{
			name:        "preserves existing GOFLAGS",
			initial:     "-mod=vendor",
			tags:        testIntegrationTag,
			wantGOFLAGS: "-tags=" + testIntegrationTag + " -mod=vendor",
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
