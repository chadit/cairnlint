package main_test

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"
)

// TestOnlyHTTPStatusLiteralRunsOnDependencies checks which analyzers run on a
// package that is only imported: those that declare facts, each one a cost on
// every run, so adding one is a decision. The dependency does not type check,
// so each analyzer that runs on it reports "analysis skipped" under that
// package in the -json output.
func TestOnlyHTTPStatusLiteralRunsOnDependencies(t *testing.T) {
	t.Parallel()

	root := writeModule(t, map[string]string{
		"dep/dep.go": "package dep\n\n// Code is a status.\nvar Code int = \"two hundred\"\n",
		"app/app.go": "package app\n\nimport \"example.com/fixture/dep\"\n\n// Use reads the code.\nfunc Use() int { return dep.Code }\n",
	})

	run := runTool(t, root, []string{agentModeOn}, "-json", "./app")

	var packages map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(run.stdout), &packages); err != nil {
		t.Fatalf("parse -json output: %v\nstdout:\n%s\nstderr:\n%s", err, run.stdout, run.stderr)
	}

	got := slices.Sorted(maps.Keys(packages["example.com/fixture/dep"]))
	if want := []string{"httpstatusliteral"}; !slices.Equal(got, want) {
		t.Errorf("analyzers run on the dependency: %v, want %v", got, want)
	}

	if _, analyzed := packages["example.com/fixture/app"]; !analyzed {
		t.Errorf("-json output has no entry for the root package:\n%s", run.stdout)
	}
}
