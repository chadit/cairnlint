package main_test

import (
	"bufio"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// readmeAnalyzerHeader is the first header cell of every README table that
// lists registered analyzers. Tables with another header, such as the skipped
// modernizers, name analyzers cairnlint does not run.
const readmeAnalyzerHeader = "Analyzer"

var (
	readmeAgentHeading    = regexp.MustCompile(`^Agent-only analyzers \((\d+)\)$`)
	readmeModernizerCount = regexp.MustCompile(`^(\d+) analyzers imported from the upstream`)
)

// readmeContent holds what the drift tests read from README.md: the analyzer
// names under each ### heading, and the stated modernizer count.
type readmeContent struct {
	tables          map[string][]string
	modernizerCount int
}

func readReadme(t *testing.T) readmeContent {
	t.Helper()

	file, err := os.Open("README.md")
	if err != nil {
		t.Fatalf("open README: %v", err)
	}

	defer func() { _ = file.Close() }()

	content := readmeContent{tables: map[string][]string{}, modernizerCount: -1}

	var heading, tableHeader string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if !strings.HasPrefix(line, "|") {
			tableHeader = ""
			heading = nextHeading(line, heading)
			content.modernizerCount = max(content.modernizerCount, leadingCount(readmeModernizerCount, line))

			continue
		}

		cell := firstCell(line)
		if tableHeader == "" {
			tableHeader = cell

			continue
		}

		if tableHeader == readmeAnalyzerHeader && !strings.HasPrefix(cell, "-") {
			content.tables[heading] = append(content.tables[heading], strings.Trim(cell, "`"))
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("read README: %v", err)
	}

	return content
}

// nextHeading returns the section a non-table line leaves the parser in. A
// level-two heading closes the current subsection.
func nextHeading(line, current string) string {
	switch {
	case strings.HasPrefix(line, "### "):
		return strings.TrimPrefix(line, "### ")
	case strings.HasPrefix(line, "## "):
		return ""
	}

	return current
}

// leadingCount returns the number pattern captures in line, or -1 when the
// line does not match.
func leadingCount(pattern *regexp.Regexp, line string) int {
	match := pattern.FindStringSubmatch(line)
	if match == nil {
		return -1
	}

	count, err := strconv.Atoi(match[1])
	if err != nil {
		return -1
	}

	return count
}

func firstCell(row string) string {
	cells := strings.Split(row, "|")
	if len(cells) < 2 {
		return ""
	}

	return strings.TrimSpace(cells[1])
}

// TestReadmeMatchesListOutput compares each README analyzer table with the
// --list section of the same name, so a rule added, removed, or moved without
// a README edit fails here. The agent section is matched by its heading count.
func TestReadmeMatchesListOutput(t *testing.T) {
	t.Parallel()

	tables := readReadme(t).tables

	for _, section := range listAnalyzers(t) {
		heading := readmeHeading(tables, section)

		got := slices.Sorted(slices.Values(tables[heading]))
		if want := slices.Sorted(slices.Values(section.analyzers)); !slices.Equal(got, want) {
			t.Errorf("README section %q lists %v\n--list section %q lists %v", heading, got, section.name, want)
		}

		delete(tables, heading)
	}

	for heading, names := range tables {
		t.Errorf("README section %q lists analyzers %v but --list has no such section", heading, names)
	}
}

// readmeHeading returns the README heading that documents section. The agent
// heading carries a count, so it must equal the number of agent analyzers.
func readmeHeading(tables map[string][]string, section listSection) string {
	if !strings.HasPrefix(section.name, agentSectionPrefix) {
		return section.name
	}

	for heading := range tables {
		if leadingCount(readmeAgentHeading, heading) == len(section.analyzers) {
			return heading
		}
	}

	return "Agent-only analyzers (" + strconv.Itoa(len(section.analyzers)) + ")"
}

// TestReadmeModernizerCount checks the count stated above the modernizer
// table, which no table row would catch on its own.
func TestReadmeModernizerCount(t *testing.T) {
	t.Parallel()

	want := -1

	for _, section := range listAnalyzers(t) {
		if strings.HasPrefix(section.name, "Modernizers") {
			want = len(section.analyzers)
		}
	}

	if got := readReadme(t).modernizerCount; got != want {
		t.Errorf("README says %d modernizers are imported, --list shows %d", got, want)
	}
}
