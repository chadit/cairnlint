package main_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// agentSummaryLine is the stderr line agent mode prints once a finding has
// gone to the agent file.
const agentSummaryLine = "[agent] heuristic findings written to "

// stubModule holds one package with a stub that only the agent tier reports.
func stubModule(t *testing.T) string {
	t.Helper()

	return writeModule(t, map[string]string{"stub/stub.go": source(stubSource, "stub")})
}

// wantAgentFinding fails t unless run put the stub finding in the agent file
// and nowhere else.
func wantAgentFinding(t *testing.T, run toolRun) {
	t.Helper()

	if !strings.Contains(run.agentFile, "stub/stub.go:4:6: "+stubMessage) {
		t.Errorf("agent file lacks the stub finding:\n%s\nstderr:\n%s", run.agentFile, run.stderr)
	}

	if strings.Contains(run.stderr, stubMessage) {
		t.Errorf("the agent finding also reached stderr:\n%s", run.stderr)
	}
}

// wantNoAgentMode fails t if run shows any sign of the agent tier. The clean
// exit code proves the run loaded the package, so a broken fixture cannot pass.
func wantNoAgentMode(t *testing.T, run toolRun) {
	t.Helper()

	if run.code != exitClean {
		t.Errorf("exit code %d, want %d; stderr:\n%s", run.code, exitClean, run.stderr)
	}

	if run.agentFile != "" || strings.Contains(run.stderr, stubMessage) || strings.Contains(run.stderr, agentSummaryLine) {
		t.Errorf("agent tier ran; agent file:\n%s\nstderr:\n%s", run.agentFile, run.stderr)
	}
}

func TestAgentFlagEnablesAgentTier(t *testing.T) {
	t.Parallel()

	root := stubModule(t)

	for _, flag := range []string{"--agent", "-agent"} {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()

			wantAgentFinding(t, runTool(t, root, nil, flag, "./stub"))
		})
	}
}

func TestAgentTierIsOffForAPersonByDefault(t *testing.T) {
	t.Parallel()

	wantNoAgentMode(t, runTool(t, stubModule(t), nil, "./stub"))
}

func TestAgentVariablesEnableAgentTier(t *testing.T) {
	t.Parallel()

	root := stubModule(t)

	for _, name := range agentVariables() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			wantAgentFinding(t, runTool(t, root, []string{name + "=1"}, "./stub"))
		})
	}
}

// TestEmptyAgentVariablesLeaveAgentTierOff covers the near miss: a variable
// that is set but empty does not count as an agent caller.
func TestEmptyAgentVariablesLeaveAgentTierOff(t *testing.T) {
	t.Parallel()

	root := stubModule(t)

	for _, name := range agentVariables() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			wantNoAgentMode(t, runTool(t, root, []string{name + "="}, "./stub"))
		})
	}
}

// TestAgentFileReport checks what a caller such as lint.sh reads: one file in
// TMPDIR named for the process, a stderr line that points at it, and an exit
// code that agent findings alone leave clean.
func TestAgentFileReport(t *testing.T) {
	t.Parallel()

	run := runTool(t, stubModule(t), []string{agentModeOn}, "./stub")
	wantAgentFinding(t, run)

	paths, err := filepath.Glob(filepath.Join(run.tmpDir, "cairnlint-agent-*.txt"))
	if err != nil || len(paths) != 1 {
		t.Fatalf("agent files in TMPDIR: %v (%v), want exactly one", paths, err)
	}

	if want := agentSummaryLine + paths[0]; !strings.Contains(run.stderr, want) {
		t.Errorf("stderr lacks %q:\n%s", want, run.stderr)
	}

	if run.code != exitClean {
		t.Errorf("exit code %d, want %d", run.code, exitClean)
	}
}

func TestAgentFileAbsentWhenClean(t *testing.T) {
	t.Parallel()

	root := writeModule(t, map[string]string{"clean/clean.go": source(cleanSource, "clean")})

	run := runTool(t, root, []string{agentModeOn}, "./clean")
	if run.agentFile != "" || strings.Contains(run.stderr, agentSummaryLine) {
		t.Errorf("a clean run left an agent file or summary; file:\n%s\nstderr:\n%s", run.agentFile, run.stderr)
	}

	if run.code != exitClean {
		t.Errorf("exit code %d, want %d", run.code, exitClean)
	}
}

// TestAgentFindingFallsBackToStderr points TMPDIR at a regular file so the
// agent file cannot be created, which fails the same way for every user. The
// go command's own scratch space moves to GOTMPDIR so loading still works.
func TestAgentFindingFallsBackToStderr(t *testing.T) {
	t.Parallel()

	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatalf("write blocker file: %v", err)
	}

	env := []string{agentModeOn, "TMPDIR=" + blocker, "GOTMPDIR=" + t.TempDir()}

	run := runTool(t, stubModule(t), env, "./stub")
	if !strings.Contains(run.stderr, "stub/stub.go:4:6: "+stubMessage) {
		t.Errorf("stderr lacks the agent finding:\n%s", run.stderr)
	}

	if strings.Contains(run.stderr, agentSummaryLine) {
		t.Errorf("stderr points at an agent file that could not be written:\n%s", run.stderr)
	}

	if run.code != exitFindings {
		t.Errorf("exit code %d, want %d", run.code, exitFindings)
	}
}

// Where the lone-implementation tests write the package and its test file.
const (
	loneImplFile  = "store/store.go"
	storeTestFile = "store/store_test.go"
)

// loneImplSource declares an interface with one implementation and no test
// double, which agentloneimpl reports at the interface on line 4.
const loneImplSource = "package store\n\n// Storer loads a record by id.\ntype Storer interface {\n" +
	"\tLoad(id string) ([]byte, error)\n}\n\ntype fileStore struct{ root string }\n\n" +
	"// Load reads the record from the configured root.\n" +
	"func (f *fileStore) Load(id string) ([]byte, error) {\n\treturn []byte(f.root + id), nil\n}\n\n" +
	"// New returns the store behind the interface.\nfunc New(root string) Storer { return &fileStore{root: root} }\n"

// TestLoneImplSurvivesTestFiles checks that a test file which holds no test
// double leaves the lone-implementation finding in place, whether it sits in
// the package or in the external test package beside it.
func TestLoneImplSurvivesTestFiles(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		testFile string
	}{
		{name: "no test file"},
		{name: "internal test file", testFile: "package store\n\nimport \"testing\"\n\nfunc TestLog(t *testing.T) { t.Log(\"internal\") }\n"},
		{name: "external test file", testFile: "package store_test\n\nimport \"testing\"\n\nfunc TestLog(t *testing.T) { t.Log(\"external\") }\n"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			files := map[string]string{loneImplFile: loneImplSource}
			if testCase.testFile != "" {
				files[storeTestFile] = testCase.testFile
			}

			run := runTool(t, writeModule(t, files), []string{agentModeOn}, "./store")
			if !strings.Contains(run.agentFile, "store/store.go:4:6: [agent] interface Storer has exactly one implementing type") {
				t.Errorf("agent file lacks the Storer finding:\n%s\nstderr:\n%s", run.agentFile, run.stderr)
			}
		})
	}
}

// loneParamSource passes the same value for strict at both call sites, which
// agentloneparam reports at render on line 4.
const loneParamSource = "package store\n\n// render formats tpl, strictly when asked.\n" +
	"func render(tpl string, strict bool) string {\n\tif strict {\n\t\treturn \"strict:\" + tpl\n\t}\n\n" +
	"\treturn tpl\n}\n\n// Pages renders both pages.\n" +
	"func Pages() []string {\n\treturn []string{render(\"a\", true), render(\"b\", true)}\n}\n"

// TestLoneParamSurvivesExternalTestFile checks that an external test file,
// which makes no in-package test variant, leaves the lone-parameter finding
// in place.
func TestLoneParamSurvivesExternalTestFile(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"store/params.go": loneParamSource,
		storeTestFile:     "package store_test\n\nimport \"testing\"\n\nfunc TestLog(t *testing.T) { t.Log(\"external\") }\n",
	}

	run := runTool(t, writeModule(t, files), []string{agentModeOn}, "./store")
	if !strings.Contains(run.agentFile, "store/params.go:4:25: [agent] parameter strict of render receives") {
		t.Errorf("agent file lacks the strict finding:\n%s\nstderr:\n%s", run.agentFile, run.stderr)
	}
}

// otherGOOSTestFile names an in-package test file whose GOOS suffix the host
// build leaves out.
func otherGOOSTestFile() string {
	if runtime.GOOS == "windows" {
		return "store/store_linux_test.go"
	}

	return "store/store_windows_test.go"
}

// TestAgentFindingsSurviveExcludedTestFiles checks that an in-package test
// file the build leaves out, by a tag the run does not set or by GOOS, does
// not hide the lone-implementation and lone-parameter findings, since the
// excluded file makes no variant that could report them.
func TestAgentFindingsSurviveExcludedTestFiles(t *testing.T) {
	t.Parallel()

	testBody := "package store\n\nimport \"testing\"\n\nfunc TestLog(t *testing.T) { t.Log(\"excluded\") }\n"
	cases := []struct {
		name     string
		fileName string
		contents string
	}{
		{name: "build tag", fileName: storeTestFile, contents: "//go:build integration\n\n" + testBody},
		{name: "other GOOS", fileName: otherGOOSTestFile(), contents: testBody},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			files := map[string]string{
				loneImplFile:      loneImplSource,
				"store/params.go": loneParamSource,
				testCase.fileName: testCase.contents,
			}

			run := runTool(t, writeModule(t, files), []string{agentModeOn}, "./store")
			for _, want := range []string{"store/store.go:4:6: [agent] interface Storer", "store/params.go:4:25: [agent] parameter strict"} {
				if !strings.Contains(run.agentFile, want) {
					t.Errorf("agent file lacks %q:\n%s\nstderr:\n%s", want, run.agentFile, run.stderr)
				}
			}
		})
	}
}

// TestLoneImplClearedByInternalTestDouble covers the near miss: a test double
// in the package's own test file is a second implementation.
func TestLoneImplClearedByInternalTestDouble(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		loneImplFile: loneImplSource,
		storeTestFile: "package store\n\nimport \"testing\"\n\ntype stubStore struct{}\n\n" +
			"func (stubStore) Load(string) ([]byte, error) { return []byte(\"stub\"), nil }\n\n" +
			"func TestStub(t *testing.T) {\n\tvar s Storer = stubStore{}\n\tif _, err := s.Load(\"x\"); err != nil {\n" +
			"\t\tt.Fatal(err)\n\t}\n}\n",
	}

	run := runTool(t, writeModule(t, files), []string{agentModeOn}, "./store")
	if strings.Contains(run.agentFile, "interface Storer") {
		t.Errorf("a test double did not clear the Storer finding:\n%s", run.agentFile)
	}

	if run.code != exitClean && run.code != exitFindings {
		t.Errorf("exit code %d; stderr:\n%s", run.code, run.stderr)
	}
}

// TestNolintSuppressesFindings checks //nolint through the binary on both
// tiers: a directive naming the rule suppresses it, and one naming another
// rule does not.
func TestNolintSuppressesFindings(t *testing.T) {
	t.Parallel()

	root := writeModule(t, map[string]string{
		"dot/named.go": "package dot\n\nimport . \"strings\" //nolint:nodotimport // kept for the test\n\nvar _ = TrimSpace\n",
		"dot/other.go": "package dot\n\nimport . \"bytes\" //nolint:emdash // names another rule\n\nvar _ = TrimSpace\n",
		"stub/stub.go": "package stub\n\n// Validate checks the input and reports the first problem it finds.\n" +
			"//\n//nolint:agentstubbody // kept for the test\nfunc Validate(input string) error {\n\treturn nil\n}\n",
	})

	run := runTool(t, root, []string{agentModeOn}, "./dot", "./stub")

	if strings.Contains(run.stderr, "dot/named.go") {
		t.Errorf("a finding under //nolint:nodotimport was reported:\n%s", run.stderr)
	}

	if !strings.Contains(run.stderr, "dot/other.go:3:8: dot import") {
		t.Errorf("a //nolint for another rule suppressed the finding:\n%s", run.stderr)
	}

	if strings.Contains(run.agentFile, "stub/stub.go") {
		t.Errorf("an agent finding under //nolint:agentstubbody was written:\n%s", run.agentFile)
	}
}
