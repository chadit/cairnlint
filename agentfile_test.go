package main_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ownAgentFile is the shell expression for the agent file path the run that
// execs next will use, since exec keeps the shell's PID.
const ownAgentFile = `"$TMPDIR/cairnlint-agent-$$.txt"`

// TestAgentFileRefusesPlantedSymlink plants a symlink at the agent file path
// before the run, as another user on a shared /tmp could. The run must leave
// the link's target alone and still report the finding, through the stderr
// fallback for a file it cannot take.
func TestAgentFileRefusesPlantedSymlink(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "victim.txt")
	if err := os.WriteFile(target, []byte("keep me\n"), 0o600); err != nil {
		t.Fatalf("write link target: %v", err)
	}

	run := runAgentToolAfter(t, "ln -s '"+target+"' "+ownAgentFile, stubModule(t), "./stub")

	after, err := os.ReadFile(target)
	if err != nil || string(after) != "keep me\n" {
		t.Errorf("the link's target changed or is unreadable (%v):\n%s", err, after)
	}

	if !strings.Contains(run.stderr, "stub/stub.go:4:6: "+stubMessage) {
		t.Errorf("stderr lacks the agent finding:\n%s", run.stderr)
	}

	if strings.Contains(run.stderr, agentSummaryLine) {
		t.Errorf("stderr points at an agent file the run refused:\n%s", run.stderr)
	}

	if run.code != exitFindings {
		t.Errorf("exit code %d, want %d", run.code, exitFindings)
	}
}

// TestAgentFileResetsLeftoverMode plants a world-readable file with old
// findings at the agent file path, as a reused PID could leave. After the run
// the file is private to its owner and holds only this run's findings.
func TestAgentFileResetsLeftoverMode(t *testing.T) {
	t.Parallel()

	before := "printf 'old.go:1:1: [agent] stale finding\\n' > " + ownAgentFile + "\nchmod 644 " + ownAgentFile

	run := runAgentToolAfter(t, before, stubModule(t), "./stub")

	paths, err := filepath.Glob(filepath.Join(run.tmpDir, "cairnlint-agent-*.txt"))
	if err != nil || len(paths) != 1 {
		t.Fatalf("agent files in TMPDIR: %v (%v), want exactly one", paths, err)
	}

	info, err := os.Stat(paths[0])
	if err != nil {
		t.Fatalf("stat agent file: %v", err)
	}

	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("agent file mode %o, want 600", mode)
	}

	if !strings.Contains(run.agentFile, "stub/stub.go:4:6: "+stubMessage) || strings.Contains(run.agentFile, "stale finding") {
		t.Errorf("agent file should hold only this run's finding:\n%s", run.agentFile)
	}
}

// wantFallback fails t unless every finding reached stderr through the
// fallback: no summary line pointing at a file, and the findings exit code.
func wantFallback(t *testing.T, run toolRun, findings ...string) {
	t.Helper()

	for _, finding := range findings {
		if !strings.Contains(run.stderr, finding) {
			t.Errorf("stderr lacks %q:\n%s", finding, run.stderr)
		}
	}

	if strings.Contains(run.stderr, agentSummaryLine) {
		t.Errorf("stderr points at an agent file the run refused:\n%s", run.stderr)
	}

	if run.code != exitFindings {
		t.Errorf("exit code %d, want %d; stderr:\n%s", run.code, exitFindings, run.stderr)
	}
}

// TestAgentFileRefusesSymlinkForEveryFinding plants a symlink and runs a
// package with two findings. The writer decides per finding, so both must
// reach stderr and neither may reach the link's target.
func TestAgentFileRefusesSymlinkForEveryFinding(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "victim.txt")
	if err := os.WriteFile(target, []byte("keep me\n"), 0o600); err != nil {
		t.Fatalf("write link target: %v", err)
	}

	root := writeModule(t, map[string]string{
		"pair/pair.go": source(stubSource, "pair") + "\n// Check checks the input and reports the first problem it finds.\n" +
			"func Check(input string) error {\n\treturn nil\n}\n",
	})

	run := runAgentToolAfter(t, "ln -s '"+target+"' "+ownAgentFile, root, "./pair")
	wantFallback(t, run, "pair/pair.go:4:6: "+stubMessage, "[agent] func Check returns without doing the work")

	if after, err := os.ReadFile(target); err != nil || string(after) != "keep me\n" {
		t.Errorf("the link's target changed or is unreadable (%v):\n%s", err, after)
	}
}

// TestAgentFileRefusesDanglingSymlink plants a symlink to a path that does
// not exist. Following it would create a file there, so nothing may appear.
func TestAgentFileRefusesDanglingSymlink(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "absent.txt")

	run := runAgentToolAfter(t, "ln -s '"+target+"' "+ownAgentFile, stubModule(t), "./stub")
	wantFallback(t, run, "stub/stub.go:4:6: "+stubMessage)

	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Errorf("the run created the dangling link's target (err %v)", err)
	}
}

// TestAgentFileRefusesPlantedFIFO plants a FIFO with no reader at the agent
// file path. Opening it to write would block forever, so the run must end on
// its own, report through the fallback, and leave the FIFO where it was.
func TestAgentFileRefusesPlantedFIFO(t *testing.T) {
	t.Parallel()

	run := runAgentToolAfter(t, "mkfifo "+ownAgentFile, stubModule(t), "./stub")
	wantFallback(t, run, "stub/stub.go:4:6: "+stubMessage)

	paths, err := filepath.Glob(filepath.Join(run.tmpDir, "cairnlint-agent-*.txt"))
	if err != nil || len(paths) != 1 {
		t.Fatalf("agent-named entries in TMPDIR: %v (%v), want the FIFO alone", paths, err)
	}

	if info, err := os.Lstat(paths[0]); err != nil || info.Mode()&os.ModeNamedPipe == 0 {
		t.Errorf("the FIFO at the agent file path is gone or replaced (err %v)", err)
	}
}

// TestAgentFileRefusesFIFOWithReader plants a FIFO and holds it open on a
// spare descriptor that survives the exec, so the tool's open succeeds. The
// writer must still refuse it, since only a regular file is an agent file.
func TestAgentFileRefusesFIFOWithReader(t *testing.T) {
	t.Parallel()

	run := runAgentToolAfter(t, "mkfifo "+ownAgentFile+"\nexec 3<> "+ownAgentFile, stubModule(t), "./stub")
	wantFallback(t, run, "stub/stub.go:4:6: "+stubMessage)

	paths, err := filepath.Glob(filepath.Join(run.tmpDir, "cairnlint-agent-*.txt"))
	if err != nil || len(paths) != 1 {
		t.Fatalf("agent-named entries in TMPDIR: %v (%v), want the FIFO alone", paths, err)
	}

	if info, err := os.Lstat(paths[0]); err != nil || info.Mode()&os.ModeNamedPipe == 0 {
		t.Errorf("the FIFO at the agent file path is gone or replaced (err %v)", err)
	}
}
