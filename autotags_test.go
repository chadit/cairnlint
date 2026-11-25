package main_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

// untaggedFile is the file every pass analyzes, whatever tags it sets.
const untaggedFile = "base.go"

// passHeader matches the line -tags=auto prints before each pass.
var passHeader = regexp.MustCompile(`(?m)^cairnlint: pass (.+)$`)

// passLabels returns the label of every pass a -tags=auto run announced, sorted.
func passLabels(stderr string) []string {
	matches := passHeader.FindAllStringSubmatch(stderr, -1)

	labels := make([]string, 0, len(matches))
	for _, match := range matches {
		labels = append(labels, match[1])
	}

	slices.Sort(labels)

	return labels
}

// TestAutoTagsRunsOnePassPerUserTag plants user tags beside GOOS, cgo, and
// Go version constraints, and tags in places the walk skips, then checks that
// only the user tags get a pass and that each tagged file is linted.
func TestAutoTagsRunsOnePassPerUserTag(t *testing.T) {
	t.Parallel()

	root := writeModule(t, map[string]string{
		untaggedFile:       source(cleanSource, "root"),
		"integration.go":   "//go:build integration\n\npackage root\n\n// Live runs \xe2\x80\x94 against a server.\nconst Live = 1\n",
		"sub/base.go":      source(cleanSource, "sub"),
		"sub/e2e.go":       "//go:build e2e && !slow\n\npackage sub\n\n// E2E runs \u2014 end to end.\nconst E2E = 1\n",
		"sub/platform.go":  "//go:build linux || darwin\n\npackage sub\n",
		"sub/cgo.go":       "//go:build cgo\n\npackage sub\n",
		"sub/version.go":   "//go:build go1.21\n\npackage sub\n",
		"sub/either.go":    "//go:build custom || integration\n\npackage sub\n",
		"legacy/base.go":   source(cleanSource, "legacy"),
		"legacy/legacy.go": "// +build legacytag\n\npackage legacy\n",
		"testdata/skip.go": "//go:build skipped\n\npackage skip\n",
		"vendor/skip.go":   "//go:build vendored\n\npackage skip\n",
		".hidden/skip.go":  "//go:build hidden\n\npackage skip\n",
		"notes.md":         "//go:build fakeone\n\nsome text\n",
	})

	run := runTool(t, root, nil, "-tags=auto", "./...")

	want := []string{"-tags=custom", "-tags=e2e", integrationTags, "-tags=legacytag", "-tags=slow", "default build"}
	if got := passLabels(run.stderr); !slices.Equal(got, want) {
		t.Errorf("passes %v, want %v; stderr:\n%s", got, want, run.stderr)
	}

	for _, finding := range []string{"integration.go:5:1: " + emDashMessage, "sub/e2e.go:5:1: " + emDashMessage} {
		if !strings.Contains(run.stderr, finding) {
			t.Errorf("stderr lacks the tagged finding %q:\n%s", finding, run.stderr)
		}
	}
}

// TestAutoTagsWithoutUserTags covers a module whose only constraints are
// reserved ones, so the default build is the only pass.
func TestAutoTagsWithoutUserTags(t *testing.T) {
	t.Parallel()

	root := writeModule(t, map[string]string{
		untaggedFile:  source(cleanSource, "root"),
		"platform.go": "//go:build linux || darwin\n\npackage root\n",
	})

	run := runTool(t, root, nil, "-tags=auto", "./...")
	if got := passLabels(run.stderr); !slices.Equal(got, []string{"default build"}) {
		t.Errorf("passes %v, want only the default build; stderr:\n%s", got, run.stderr)
	}

	if run.code != exitClean {
		t.Errorf("exit code %d, want %d; stderr:\n%s", run.code, exitClean, run.stderr)
	}
}

// TestAutoTagsFailedPassDecidesExitCode checks that a pass that fails to load
// sets the exit code even when the other passes report findings, so a broken
// build configuration is never read as an ordinary lint failure.
func TestAutoTagsFailedPassDecidesExitCode(t *testing.T) {
	t.Parallel()

	root := writeModule(t, map[string]string{
		untaggedFile: source(emDashSource, "root"),
		"broken.go":  "//go:build broken\n\npackage root\nfunc {\n",
	})

	run := runTool(t, root, nil, "-tags=auto", "./...")
	if run.code != exitFailure {
		t.Errorf("exit code %d, want %d from the failed pass; stderr:\n%s", run.code, exitFailure, run.stderr)
	}

	if !strings.Contains(run.stderr, untaggedFile+":3:1: "+emDashMessage) {
		t.Errorf("the default pass's finding is missing; stderr:\n%s", run.stderr)
	}
}

// TestAutoTagsPrintsRepeatedFindingOnce checks that a finding in an untagged
// file, which every pass analyzes, is printed once.
func TestAutoTagsPrintsRepeatedFindingOnce(t *testing.T) {
	t.Parallel()

	root := writeModule(t, map[string]string{
		untaggedFile: source(emDashSource, "root"),
		"alpha.go":   "//go:build alpha\n\npackage root\n",
		"beta.go":    "//go:build beta\n\npackage root\n",
	})

	run := runTool(t, root, nil, "-tags=auto", "./...")
	if got := len(passLabels(run.stderr)); got != 3 {
		t.Fatalf("%d passes, want 3; stderr:\n%s", got, run.stderr)
	}

	if got := strings.Count(run.stderr, untaggedFile+":3:1: "+emDashMessage); got != 1 {
		t.Errorf("the finding printed %d times, want once; stderr:\n%s", got, run.stderr)
	}

	if run.code != exitFindings {
		t.Errorf("exit code %d, want %d", run.code, exitFindings)
	}
}
