package main_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Agent file names that no run here can write itself, since no test process
// gets a PID this small.
const (
	oldAgentFile   = "cairnlint-agent-1.txt"
	newAgentFile   = "cairnlint-agent-2.txt"
	agentNamedDir  = "cairnlint-agent-3.txt"
	agentNamedLink = "cairnlint-agent-4.txt"
	notesFile      = "cairnlint-agent-notes.txt"
	backupFile     = "cairnlint-agent-2.txt.bak"
	noPIDFile      = "cairnlint-agent-.txt"
)

// plantedTmp is a TMPDIR holding one entry of each kind the sweep has to
// judge: newAgentFile 23 hours old, the rest 25 hours old, either side of the
// one-day limit.
type plantedTmp struct {
	dir        string
	linkTarget string
}

// plantAgentFiles builds a plantedTmp. The symlink points at an old regular
// file outside TMPDIR, so a sweep that followed links would see an old file.
// The link itself keeps a fresh time: the standard library cannot date a link
// without following it.
func plantAgentFiles(t *testing.T) plantedTmp {
	t.Helper()

	planted := plantedTmp{dir: t.TempDir(), linkTarget: filepath.Join(t.TempDir(), "target.txt")}

	for _, name := range []string{oldAgentFile, newAgentFile, notesFile, backupFile, noPIDFile} {
		if err := os.WriteFile(filepath.Join(planted.dir, name), []byte("x.go:1:1: [agent] old finding\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	if err := os.WriteFile(planted.linkTarget, []byte("not an agent file\n"), 0o600); err != nil {
		t.Fatalf("write link target: %v", err)
	}

	if err := os.Mkdir(filepath.Join(planted.dir, agentNamedDir), 0o700); err != nil {
		t.Fatalf("create directory: %v", err)
	}

	if err := os.Symlink(planted.linkTarget, filepath.Join(planted.dir, agentNamedLink)); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	dateEntry(t, filepath.Join(planted.dir, newAgentFile), 23*time.Hour)

	// Chtimes follows the symlink, so this dates its target.
	for _, name := range []string{oldAgentFile, notesFile, backupFile, noPIDFile, agentNamedDir, agentNamedLink} {
		dateEntry(t, filepath.Join(planted.dir, name), 25*time.Hour)
	}

	return planted
}

// dateEntry sets path's access and modification times to age ago.
func dateEntry(t *testing.T, path string, age time.Duration) {
	t.Helper()

	stamp := time.Now().Add(-age)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatalf("date %s: %v", path, err)
	}
}

// wantPresent fails t unless every name exists in dir.
func wantPresent(t *testing.T, dir string, names ...string) {
	t.Helper()

	for _, name := range names {
		if _, err := os.Lstat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s is gone from TMPDIR: %v", name, err)
		}
	}
}

// summaryOnlyStderr matches stderr that holds nothing but the agent summary
// line naming this run's file in dir, and captures that file's path.
func summaryOnlyStderr(dir string) *regexp.Regexp {
	return regexp.MustCompile(`\A` + regexp.QuoteMeta(agentSummaryLine) +
		`(` + regexp.QuoteMeta(filepath.Join(dir, "cairnlint-agent-")) + `[0-9]+\.txt)\n\z`)
}

// TestAgentRunSweepsDayOldAgentFiles checks that an agent-mode run removes a
// leftover agent file more than a day old, whether or not it has findings,
// and that the sweep prints nothing: stderr is exactly what the run prints
// without one.
func TestAgentRunSweepsDayOldAgentFiles(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		source   string
		findings bool
	}{
		{name: "run with findings", source: stubSource, findings: true},
		{name: "clean run", source: cleanSource},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			planted := plantAgentFiles(t)
			root := writeModule(t, map[string]string{"pkg/pkg.go": source(testCase.source, "pkg")})

			run := runTool(t, root, []string{agentModeOn, "TMPDIR=" + planted.dir}, "./pkg")
			if run.code != exitClean {
				t.Fatalf("exit code %d, want %d; stderr:\n%s", run.code, exitClean, run.stderr)
			}

			if _, err := os.Lstat(filepath.Join(planted.dir, oldAgentFile)); !os.IsNotExist(err) {
				t.Errorf("%s older than a day is still in TMPDIR (err %v)", oldAgentFile, err)
			}

			if !testCase.findings {
				if run.stderr != "" {
					t.Errorf("a clean run printed to stderr:\n%s", run.stderr)
				}

				return
			}

			wantFindingInOwnFile(t, planted.dir, run.stderr)
		})
	}
}

// wantFindingInOwnFile fails t unless stderr is only the summary line and
// the file it names holds the stub finding.
func wantFindingInOwnFile(t *testing.T, dir, stderr string) {
	t.Helper()

	match := summaryOnlyStderr(dir).FindStringSubmatch(stderr)
	if match == nil {
		t.Fatalf("stderr is not exactly the agent summary line for %s:\n%s", dir, stderr)
	}

	contents, err := os.ReadFile(match[1])
	if err != nil {
		t.Fatalf("read the run's agent file: %v", err)
	}

	if !strings.Contains(string(contents), "pkg/pkg.go:4:6: "+stubMessage) {
		t.Errorf("the run's agent file lacks the stub finding:\n%s", contents)
	}
}

// TestAgentRunKeepsOtherFiles covers the near misses: a matching file 23
// hours old, old files whose names only resemble an agent file, and an old
// directory or symlink with a matching name all survive, and so does the
// link's target. The run's own agent file still appears with its summary line.
func TestAgentRunKeepsOtherFiles(t *testing.T) {
	t.Parallel()

	planted := plantAgentFiles(t)

	run := runTool(t, stubModule(t), []string{agentModeOn, "TMPDIR=" + planted.dir}, "./stub")
	wantPresent(t, planted.dir, newAgentFile, notesFile, backupFile, noPIDFile, agentNamedDir, agentNamedLink)
	wantPresent(t, filepath.Dir(planted.linkTarget), filepath.Base(planted.linkTarget))

	match := summaryOnlyStderr(planted.dir).FindStringSubmatch(run.stderr)
	if match == nil {
		t.Fatalf("stderr is not exactly the agent summary line for %s:\n%s", planted.dir, run.stderr)
	}

	wantPresent(t, planted.dir, filepath.Base(match[1]))
}

// TestNonAgentRunKeepsOldAgentFiles checks that a run outside agent mode
// leaves even an old agent file alone.
func TestNonAgentRunKeepsOldAgentFiles(t *testing.T) {
	t.Parallel()

	planted := plantAgentFiles(t)

	run := runTool(t, stubModule(t), []string{"TMPDIR=" + planted.dir}, "./stub")
	if run.code != exitClean {
		t.Fatalf("exit code %d, want %d; stderr:\n%s", run.code, exitClean, run.stderr)
	}

	wantPresent(t, planted.dir, oldAgentFile)
}
