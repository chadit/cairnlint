package main_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// The exit codes a caller of cairnlint can see.
const (
	exitClean    = 0
	exitFailure  = 1
	exitUsage    = 2
	exitFindings = 3
)

// Sources the tests plant in a temporary module. Each one triggers exactly
// one rule, or none, so a case can name the finding it expects.
const (
	cleanSource  = "package %s\n\n// Name names the package.\nconst Name = %q\n"
	emDashSource = "package %s\n\n// Name names the package \u2014 for the test.\nconst Name = %q\n"
	stubSource   = "package %s\n\n// Validate checks the input and reports the first problem it finds.\n" +
		"func Validate(input string) error {\n\treturn nil\n}\n\n// Name names the package.\nconst Name = %q\n"
	emDashMessage = "em dash in comment"
	stubMessage   = "[agent] func Validate returns without doing the work"
)

// Arguments and variables several tests share.
const (
	agentModeOn     = "CAIRNLINT_AGENT=1"
	integrationTags = "-tags=integration"
)

// binaryPath is the cairnlint binary TestMain builds, so every test runs the
// tool the way a user does rather than calling into package main.
var binaryPath string //nolint:gochecknoglobals // written once by TestMain before any test runs

func TestMain(m *testing.M) {
	os.Exit(runWithBinary(m))
}

func runWithBinary(m *testing.M) int {
	dir, err := os.MkdirTemp("", "cairnlint-bin-")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "E: create binary dir: %v\n", err)

		return exitFailure
	}

	defer func() { _ = os.RemoveAll(dir) }()

	binaryPath = filepath.Join(dir, "cairnlint")

	build := exec.CommandContext(context.Background(), "go", "build", "-o", binaryPath, ".")
	build.Stderr = os.Stderr

	if err := build.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "E: build cairnlint: %v\n", err)

		return exitFailure
	}

	return m.Run()
}

// agentVariables are the variables that switch cairnlint into agent mode.
// lint.sh's CAIRNLINT_AGENT_VARS mirrors the same list.
func agentVariables() []string {
	return []string{
		"AI_AGENT", "AGENT", "CLAUDECODE", "CODEX_SANDBOX", "CODEX_THREAD_ID", "GEMINI_CLI", "CURSOR_AGENT",
		"QWEN_CODE", "GOOSE_TERMINAL", "CLINE_ACTIVE", "AUGMENT_AGENT", "TRAE_AI_SHELL_ID", "OPENCODE_CLIENT",
		"CAIRNLINT_AGENT",
	}
}

// callerEnvironment is the test's environment with every agent variable
// removed, because the suite itself often runs under an agent and each case
// has to choose the mode it tests.
func callerEnvironment() []string {
	agents := agentVariables()

	return slices.DeleteFunc(os.Environ(), func(entry string) bool {
		name, _, _ := strings.Cut(entry, "=")

		return slices.Contains(agents, name)
	})
}

// toolRun is what one cairnlint invocation left for its caller to see.
type toolRun struct {
	stdout    string
	stderr    string
	agentFile string
	tmpDir    string
	code      int
}

// runTool runs cairnlint from dir the way lint.sh's go_cairnlint does, with
// TMPDIR at a fresh directory. env is applied last, so it can replace TMPDIR.
func runTool(t *testing.T, dir string, env []string, args ...string) toolRun {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), binaryPath, args...)
	cmd.Dir = dir

	return runCommand(t, cmd, env)
}

// runAgentToolAfter runs cairnlint in agent mode from dir like runTool, after
// a shell script that then execs it. The script sees the run's TMPDIR, and
// $$ there is cairnlint's PID, so it can plant an entry at the agent file path.
func runAgentToolAfter(t *testing.T, before, dir string, args ...string) toolRun {
	t.Helper()

	script := []string{"-c", before + "\nexec \"$@\"", "sh", binaryPath}

	// A planted entry can make a broken writer block forever, so the run gets
	// a deadline and fails instead of hanging the suite.
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, "sh", slices.Concat(script, args)...)
	cmd.Dir = dir
	cmd.WaitDelay = 5 * time.Second

	return runCommand(t, cmd, []string{agentModeOn})
}

// runCommand runs cmd with TMPDIR at a fresh directory and env applied last,
// so env can replace TMPDIR, and collects what the run left.
func runCommand(t *testing.T, cmd *exec.Cmd, env []string) toolRun {
	t.Helper()

	tmpDir := t.TempDir()

	cmd.Env = slices.Concat(callerEnvironment(), []string{"TMPDIR=" + tmpDir}, env)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitErr, exited := errors.AsType[*exec.ExitError](err)
	if err != nil && !exited {
		t.Fatalf("start cairnlint: %v", err)
	}

	code := exitClean
	if exited {
		code = exitErr.ExitCode()
	}

	return toolRun{
		code:      code,
		stdout:    stdout.String(),
		stderr:    stderr.String(),
		agentFile: readAgentFiles(t, tmpDir),
		tmpDir:    tmpDir,
	}
}

// readAgentFiles returns the contents of every agent findings file in dir.
// A -tags=auto run leaves one per pass, since each pass is its own process.
func readAgentFiles(t *testing.T, dir string) string {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(dir, "cairnlint-agent-*.txt"))
	if err != nil {
		t.Fatalf("glob agent files: %v", err)
	}

	contents := make([]string, 0, len(paths))

	for _, path := range paths {
		contents = append(contents, readRegularFile(t, path))
	}

	return strings.Join(contents, "")
}

// readRegularFile returns path's contents, or "" when a test planted
// something other than a regular file there: reading a FIFO with no writer
// would block forever.
func readRegularFile(t *testing.T, path string) string {
	t.Helper()

	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return ""
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read agent file: %v", err)
	}

	return string(data)
}

// writeModule writes files, keyed by slash-separated path, into a fresh module
// and returns its root.
func writeModule(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	files["go.mod"] = "module example.com/fixture\n\ngo 1.26\n"

	for relPath, contents := range files {
		full := filepath.Join(root, filepath.FromSlash(relPath))

		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("create %s: %v", filepath.Dir(full), err)
		}

		if err := os.WriteFile(full, []byte(contents), 0o600); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}

	return root
}

// source fills one of the planted sources in for package pkg.
func source(format, pkg string) string {
	return fmt.Sprintf(format, pkg, pkg)
}

func TestExitCodes(t *testing.T) {
	t.Parallel()

	root := writeModule(t, map[string]string{
		"clean/clean.go": source(cleanSource, "clean"),
		"dash/dash.go":   source(emDashSource, "dash"),
		"broken/x.go":    "package broken\nfunc {\n",
	})

	cases := []struct {
		name       string
		wantStderr string
		args       []string
		wantCode   int
	}{
		{name: "clean package", args: []string{"./clean"}, wantCode: exitClean},
		{name: "package with a finding", args: []string{"./dash"}, wantCode: exitFindings, wantStderr: emDashMessage},
		{name: "unparsable file", args: []string{"./broken"}, wantCode: exitFailure, wantStderr: "expected 'IDENT'"},
		{name: "missing package", args: []string{"./nope"}, wantCode: exitFailure, wantStderr: "directory not found"},
		{
			name: "unknown flag", args: []string{"--bogus", "./clean"},
			wantCode: exitUsage, wantStderr: "flag provided but not defined: -bogus",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			run := runTool(t, root, nil, testCase.args...)
			if run.code != testCase.wantCode {
				t.Errorf("exit code %d, want %d; stderr:\n%s", run.code, testCase.wantCode, run.stderr)
			}

			if !strings.Contains(run.stderr, testCase.wantStderr) {
				t.Errorf("stderr lacks %q:\n%s", testCase.wantStderr, run.stderr)
			}
		})
	}
}

func TestEditFlagsAreRejected(t *testing.T) {
	t.Parallel()

	for _, flag := range []string{"-fix", "--fix", "-diff", "--diff"} {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()

			original := source(emDashSource, "dash")
			root := writeModule(t, map[string]string{"dash/dash.go": original})

			run := runTool(t, root, nil, flag, "./dash")
			if run.code != exitUsage {
				t.Errorf("exit code %d, want %d", run.code, exitUsage)
			}

			if want := flag[strings.LastIndex(flag, "-"):] + " is not supported"; !strings.Contains(run.stderr, want) {
				t.Errorf("stderr lacks %q:\n%s", want, run.stderr)
			}

			after, err := os.ReadFile(filepath.Join(root, "dash", "dash.go"))
			if err != nil || string(after) != original {
				t.Errorf("source changed or unreadable after %s: %v\n%s", flag, err, after)
			}
		})
	}
}

// TestPackageNamedFixIsNotAnEditFlag covers the near miss: a package path
// that contains "fix" is linted, not turned away.
func TestPackageNamedFixIsNotAnEditFlag(t *testing.T) {
	t.Parallel()

	root := writeModule(t, map[string]string{"fix/fix.go": source(emDashSource, "fix")})

	run := runTool(t, root, nil, "./fix")
	if run.code != exitFindings || !strings.Contains(run.stderr, "fix/fix.go:3:1: "+emDashMessage) {
		t.Errorf("exit code %d, want %d with the em dash finding; stderr:\n%s", run.code, exitFindings, run.stderr)
	}
}

// TestJSONOutputCarriesNoSuggestedFixes checks that a finding from an upstream
// analyzer that offers a fix reaches the user without the edit.
func TestJSONOutputCarriesNoSuggestedFixes(t *testing.T) {
	t.Parallel()

	root := writeModule(t, map[string]string{
		"box/box.go": "package box\n\n// Box holds anything.\ntype Box struct {\n\tValue interface{}\n}\n",
	})

	// The control: go fix runs the same upstream analyzer and offers the edit,
	// so an empty fix list from cairnlint means it stripped one, not that
	// upstream stopped offering it.
	fix := exec.CommandContext(t.Context(), "go", "fix", "-diff", "./box")
	fix.Dir = root
	fix.Env = callerEnvironment()

	// go fix -diff exits non-zero whenever the diff is not empty.
	diff, _ := fix.CombinedOutput()
	if !strings.Contains(string(diff), "+\tValue any") {
		t.Fatalf("go fix -diff offers no any fix, so this test proves nothing:\n%s", diff)
	}

	run := runTool(t, root, nil, "-json", "./box")
	if !strings.Contains(run.stdout, "interface{} can be replaced by any") {
		t.Fatalf("JSON output lacks the any finding:\n%s", run.stdout)
	}

	if strings.Contains(run.stdout, "suggested_fixes") {
		t.Errorf("JSON output carries a suggested fix:\n%s", run.stdout)
	}
}

func TestListFlag(t *testing.T) {
	t.Parallel()

	for _, flag := range []string{"--list", "-list", "--linters", "-linters"} {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()

			run := runTool(t, t.TempDir(), nil, flag)
			if run.code != exitClean {
				t.Errorf("exit code %d, want %d; stderr:\n%s", run.code, exitClean, run.stderr)
			}

			for _, want := range []string{"Available linters:", "  emdash ", "Agent-only", "  agentstubbody "} {
				if !strings.Contains(run.stdout, want) {
					t.Errorf("output lacks %q:\n%s", want, run.stdout)
				}
			}
		})
	}
}

// TestListFlagNearMiss covers a flag that only starts like --list.
func TestListFlagNearMiss(t *testing.T) {
	t.Parallel()

	run := runTool(t, t.TempDir(), nil, "--listing")
	if run.code != exitUsage || strings.Contains(run.stdout, "Available linters:") {
		t.Errorf("exit code %d, want %d and no catalog; stdout:\n%s", run.code, exitUsage, run.stdout)
	}
}

// gatedPattern names the package taggedModule writes.
const gatedPattern = "./gated"

// taggedModule holds one package whose only file needs the integration tag.
func taggedModule(t *testing.T) string {
	t.Helper()

	return writeModule(t, map[string]string{
		"gated/gated.go": "//go:build integration\n\n" + source(emDashSource, "gated"),
	})
}

func TestTagsFlag(t *testing.T) {
	t.Parallel()

	root := taggedModule(t)

	cases := []struct {
		name string
		args []string
	}{
		{name: "single dash equals", args: []string{integrationTags}},
		{name: "double dash equals", args: []string{"--tags=integration"}},
		{name: "single dash space separated", args: []string{"-tags", "integration"}},
		{name: "double dash space separated", args: []string{"--tags", "integration"}},
		{name: "comma list", args: []string{"-tags=e2e,integration"}},
		{name: "last value wins", args: []string{"-tags=other", integrationTags}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			run := runTool(t, root, nil, append(testCase.args, gatedPattern)...)
			if run.code != exitFindings || !strings.Contains(run.stderr, "gated/gated.go:5:1: "+emDashMessage) {
				t.Errorf("exit code %d, want %d with the gated finding; stderr:\n%s", run.code, exitFindings, run.stderr)
			}
		})
	}
}

func TestTagsFlagExcludesGatedPackage(t *testing.T) {
	t.Parallel()

	root := taggedModule(t)

	cases := []struct {
		name string
		args []string
	}{
		{name: "no tags flag", args: []string{gatedPattern}},
		{name: "other tag", args: []string{"-tags=other", gatedPattern}},
		{name: "space form without a value", args: []string{gatedPattern, "-tags"}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			run := runTool(t, root, nil, testCase.args...)
			if run.code != exitFailure || !strings.Contains(run.stderr, "build constraints exclude all Go files") {
				t.Errorf("exit code %d, want %d with the exclusion error; stderr:\n%s", run.code, exitFailure, run.stderr)
			}
		})
	}
}

// TestTagsFlagKeepsGOFLAGS checks that -tags adds to the caller's GOFLAGS
// rather than replacing them: a valid flag still loads, and an invalid one
// still reaches the go command.
func TestTagsFlagKeepsGOFLAGS(t *testing.T) {
	t.Parallel()

	root := taggedModule(t)

	loaded := runTool(t, root, []string{"GOFLAGS=-mod=mod"}, integrationTags, gatedPattern)
	if loaded.code != exitFindings || !strings.Contains(loaded.stderr, emDashMessage) {
		t.Errorf("with GOFLAGS=-mod=mod: exit code %d, want %d with the finding; stderr:\n%s",
			loaded.code, exitFindings, loaded.stderr)
	}

	rejected := runTool(t, root, []string{"GOFLAGS=-mod=bogus"}, integrationTags, gatedPattern)
	if rejected.code != exitFailure || !strings.Contains(rejected.stderr, "-mod=bogus not supported") {
		t.Errorf("with GOFLAGS=-mod=bogus: exit code %d, want %d naming the bad flag; stderr:\n%s",
			rejected.code, exitFailure, rejected.stderr)
	}
}

// TestTagsFlagBeatsGOFLAGSTags checks that the command line wins over a -tags
// value in GOFLAGS, as it does for the go command itself.
func TestTagsFlagBeatsGOFLAGSTags(t *testing.T) {
	t.Parallel()

	run := runTool(t, taggedModule(t), []string{"GOFLAGS=-tags=other"}, integrationTags, gatedPattern)
	if run.code != exitFindings || !strings.Contains(run.stderr, "gated/gated.go:5:1: "+emDashMessage) {
		t.Errorf("exit code %d, want %d with the gated finding; stderr:\n%s", run.code, exitFindings, run.stderr)
	}
}

// TestGOFLAGSTagsApplyWithoutTagsFlag covers the near miss: with no -tags on
// the command line, the GOFLAGS value still selects the build.
func TestGOFLAGSTagsApplyWithoutTagsFlag(t *testing.T) {
	t.Parallel()

	run := runTool(t, taggedModule(t), []string{"GOFLAGS=-tags=integration"}, gatedPattern)
	if run.code != exitFindings || !strings.Contains(run.stderr, "gated/gated.go:5:1: "+emDashMessage) {
		t.Errorf("exit code %d, want %d with the gated finding; stderr:\n%s", run.code, exitFindings, run.stderr)
	}
}
