package analyzers_test

import (
	"bufio"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"

	"github.com/chadit/cairnlint/analyzers"
)

// readmeAnalyzerHeader is the first header cell of every README table that
// lists registered analyzers. Tables with another header, such as the skipped
// modernizers, name analyzers cairnlint does not run.
const readmeAnalyzerHeader = "Analyzer"

var (
	agentHeadingPattern    = regexp.MustCompile(`^Agent-only analyzers \((\d+)\)$`)
	modernizerCountPattern = regexp.MustCompile(`^(\d+) analyzers imported from the upstream`)
)

// readmeTables maps each ### heading in README.md to the analyzer names in the
// first column of its analyzer tables, in order.
type readmeTables map[string][]string

// readmeContent holds what the drift tests read from README.md.
type readmeContent struct {
	tables          readmeTables
	modernizerCount int
}

func readReadme(t *testing.T) readmeContent {
	t.Helper()

	file, err := os.Open("../README.md")
	if err != nil {
		t.Fatalf("open README: %v", err)
	}

	defer func() { _ = file.Close() }()

	content := readmeContent{tables: readmeTables{}, modernizerCount: -1}

	var heading, tableHeader string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if !strings.HasPrefix(line, "|") {
			tableHeader = ""
			heading = nextHeading(line, heading)
			content.modernizerCount = max(content.modernizerCount, leadingCount(modernizerCountPattern, line))

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

func sortedNames(set []*analysis.Analyzer) []string {
	names := make([]string, len(set))
	for idx, analyzer := range set {
		names[idx] = analyzer.Name
	}

	slices.Sort(names)

	return names
}

// TestReadmeListsEveryRegisteredAnalyzer compares each README analyzer table
// with the category of the same name in Categories, so a rule added, removed,
// or moved without a README edit fails here instead of drifting.
func TestReadmeListsEveryRegisteredAnalyzer(t *testing.T) {
	t.Parallel()

	tables := readReadme(t).tables

	for _, category := range analyzers.Categories() {
		got := slices.Sorted(slices.Values(tables[category.Name]))
		if want := sortedNames(category.Analyzers); !slices.Equal(got, want) {
			t.Errorf("README section %q lists %v\nregistered: %v", category.Name, got, want)
		}

		delete(tables, category.Name)
	}

	for heading, names := range tables {
		if agentHeadingPattern.MatchString(heading) {
			continue
		}

		t.Errorf("README section %q lists analyzers %v but matches no category", heading, names)
	}
}

// TestReadmeListsEveryAgentAnalyzer checks the agent-only table and the count
// in its heading against AgentOnly.
func TestReadmeListsEveryAgentAnalyzer(t *testing.T) {
	t.Parallel()

	want := sortedNames(analyzers.AgentOnly())

	var found bool

	for heading, names := range readReadme(t).tables {
		count := leadingCount(agentHeadingPattern, heading)
		if count < 0 {
			continue
		}

		found = true

		if count != len(want) {
			t.Errorf("README heading %q counts %d agent analyzers, AgentOnly registers %d", heading, count, len(want))
		}

		if got := slices.Sorted(slices.Values(names)); !slices.Equal(got, want) {
			t.Errorf("README agent table lists %v\nregistered: %v", got, want)
		}
	}

	if !found {
		t.Error("README has no \"Agent-only analyzers (N)\" section")
	}
}

// TestReadmeModernizerCount checks the count stated above the modernizer
// table, which no table row would catch on its own.
func TestReadmeModernizerCount(t *testing.T) {
	t.Parallel()

	var want int

	for _, category := range analyzers.Categories() {
		if strings.HasPrefix(category.Name, "Modernizers") {
			want = len(category.Analyzers)
		}
	}

	if got := readReadme(t).modernizerCount; got != want {
		t.Errorf("README says %d modernizers are imported, Categories registers %d", got, want)
	}
}
