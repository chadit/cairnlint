package analyzers_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chadit/cairnlint/analyzers"
)

// agentFileFindings runs the contextbackground analyzer behind the agent-file
// wrapper over the named fixture, then returns the agent files the run left in
// the temp directory. The caller must point TMPDIR at a private directory.
func agentFileFindings(t *testing.T, fixture string) []string {
	t.Helper()

	wrapped := analyzers.WrapForAgentFile([]*analysis.Analyzer{findAnalyzer("contextbackground")})
	analysistest.Run(t, analysistest.TestData(), wrapped[0], fixture)

	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "*-agent-*.txt"))
	if err != nil {
		t.Fatalf("glob agent files: %v", err)
	}

	return matches
}

// readAgentFile returns the lines of the single agent file in matches.
func readAgentFile(t *testing.T, matches []string) []string {
	t.Helper()

	if len(matches) != 1 {
		t.Fatalf("agent files in temp dir: %v, want exactly 1", matches)
	}

	content, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read agent file: %v", err)
	}

	return strings.Split(strings.TrimSpace(string(content)), "\n")
}

// TestAgentFileHonorsFilters verifies that diagnostics routed to the agent
// file still pass through the nolint and generated-file filters, and that each
// one lands as its own line. The fixture has four context.Background() calls:
// one under a //nolint, one in a file with a generated header, and two plain.
// Only the plain ones may land in the file.
func TestAgentFileHonorsFilters(t *testing.T) {
	// t.Setenv rules out t.Parallel; the private temp dir keeps the
	// per-process agent file out of the shared TMPDIR.
	t.Setenv("TMPDIR", t.TempDir())

	lines := readAgentFile(t, agentFileFindings(t, "agentoutput"))
	if len(lines) != 2 {
		t.Fatalf("agent file has %d findings, want exactly 2:\n%s", len(lines), strings.Join(lines, "\n"))
	}

	for idx, want := range []string{"example_test.go:17:", "example_test.go:22:"} {
		if !strings.Contains(lines[idx], want) {
			t.Errorf("agent file line %d is %q, want the unsuppressed call at %s", idx+1, lines[idx], want)
		}
	}
}

// TestAgentFileAbsentWithoutFindings verifies a run whose agent tier reports
// nothing leaves no file behind, so clean runs do not pile empty files up in
// the temp directory. The fixture's only finding is suppressed by //nolint.
func TestAgentFileAbsentWithoutFindings(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	if matches := agentFileFindings(t, "agentoutputclean"); len(matches) != 0 {
		t.Fatalf("a run with no findings left agent files %v, want none", matches)
	}
}

// TestAgentFileReplacesStaleFile verifies the first finding truncates a file
// left by an earlier process that had the same PID, so the reader sees only
// this run's findings.
func TestAgentFileReplacesStaleFile(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	stale := filepath.Join(os.TempDir(), fmt.Sprintf("cairnlint-agent-%d.txt", os.Getpid()))
	if err := os.WriteFile(stale, []byte("stale.go:1:1: from an earlier run\n"), 0o600); err != nil {
		t.Fatalf("write stale agent file: %v", err)
	}

	lines := readAgentFile(t, agentFileFindings(t, "agentoutput"))
	for _, line := range lines {
		if strings.Contains(line, "stale.go") {
			t.Fatalf("agent file kept a stale finding: %q", line)
		}
	}

	if len(lines) != 2 {
		t.Fatalf("agent file has %d findings, want this run's 2:\n%s", len(lines), strings.Join(lines, "\n"))
	}
}
