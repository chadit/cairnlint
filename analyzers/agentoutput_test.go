package analyzers_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/chadit/cairnlint/analyzers"
)

// TestAgentFileHonorsFilters verifies that diagnostics routed to the agent
// file still pass through the nolint and generated-file filters. The fixture
// has three context.Background() calls: one under a //nolint, one in a file
// with a generated header, and one plain. Only the plain one may land in
// the file.
func TestAgentFileHonorsFilters(t *testing.T) {
	// t.Setenv rules out t.Parallel; the private temp dir keeps the
	// per-process agent file out of the shared TMPDIR.
	t.Setenv("TMPDIR", t.TempDir())

	wrapped := analyzers.WrapForAgentFile([]*analysis.Analyzer{findAnalyzer("contextbackground")})

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, wrapped[0], "agentoutput")

	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "*-agent-*.txt"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("agent files in temp dir: %v (err %v), want exactly 1", matches, err)
	}

	content, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read agent file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 1 {
		t.Fatalf("agent file has %d findings, want exactly 1:\n%s", len(lines), content)
	}

	if !strings.Contains(lines[0], "example_test.go:17:") {
		t.Fatalf("agent file finding anchors to %q, want the unsuppressed call on example_test.go:17", lines[0])
	}
}
