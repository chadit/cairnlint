package main_test

import (
	"strings"
	"testing"
)

// agentSectionPrefix starts the --list heading of the agent-only analyzers.
const agentSectionPrefix = "Agent-only"

// listSection is one heading of the --list catalog and the analyzer names
// listed under it, in order.
type listSection struct {
	name      string
	analyzers []string
}

// listAnalyzers runs `cairnlint --list` and parses its catalog, so tests read
// the analyzer set a user sees rather than the registry behind it.
func listAnalyzers(t *testing.T) []listSection {
	t.Helper()

	run := runTool(t, t.TempDir(), nil, "--list")
	if run.code != exitClean {
		t.Fatalf("--list exit code %d; stderr:\n%s", run.code, run.stderr)
	}

	var sections []listSection

	for line := range strings.Lines(run.stdout) {
		sections = addListLine(sections, strings.TrimRight(line, "\n"))
	}

	if len(sections) == 0 {
		t.Fatalf("--list printed no sections:\n%s", run.stdout)
	}

	return sections
}

// addListLine folds one catalog line into sections. A heading ends in a colon,
// and an indented row starts with the analyzer name.
func addListLine(sections []listSection, line string) []listSection {
	if name, isHeading := strings.CutSuffix(line, ":"); isHeading && !strings.HasPrefix(line, " ") {
		if name == "Available linters" {
			return sections
		}

		return append(sections, listSection{name: name})
	}

	fields := strings.Fields(line)
	if len(sections) == 0 || !strings.HasPrefix(line, "  ") || len(fields) == 0 {
		return sections
	}

	last := &sections[len(sections)-1]
	last.analyzers = append(last.analyzers, fields[0])

	return sections
}

// allListed returns every analyzer name in sections and the set of those the
// agent-only section lists.
func allListed(sections []listSection) ([]string, map[string]bool) {
	var names []string

	agents := make(map[string]bool)

	for _, section := range sections {
		names = append(names, section.analyzers...)

		if !strings.HasPrefix(section.name, agentSectionPrefix) {
			continue
		}

		for _, name := range section.analyzers {
			agents[name] = true
		}
	}

	return names, agents
}
