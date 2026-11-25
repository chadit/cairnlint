package main_test

import (
	"encoding/json"
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// updateEval re-records the flag lines of the eval file from the current
// binary. `make eval-update` sets it; every other run compares.
var updateEval = flag.Bool("update", false, "rewrite the flag lines of "+evalFile+" from this run") //nolint:gochecknoglobals // test flag

const (
	evalFile   = "testdata/eval.txt"
	evalHeader = `# What cairnlint reports over analyzers/testdata, by file, line, and rule.
# Paths are relative to analyzers/testdata/src.
# "flag": a finding the tool must report; make eval-update rewrites these.
# "quiet": a hand-written near miss; its rule reports nothing in the declaration holding it.
# Every listed analyzer needs one line of each kind.
`
	flagKind  = "flag"
	quietKind = "quiet"
)

// agentLine splits one agent findings file line into its position fields.
var agentLine = regexp.MustCompile(`^(.+):(\d+):\d+: `)

// expectation is one line of the eval file.
type expectation struct {
	kind     string
	position string
	rule     string
}

func (exp expectation) String() string {
	return exp.position + " " + exp.rule
}

// corpus is one directory the eval runs the tool from, with the environment
// that loads it.
type corpus struct {
	dir string
	env []string
}

// evalCorpora returns the directory eval positions are relative to and the
// runs the eval makes. The fixtures load in GOPATH mode, as analysistest loads
// them, since they import each other by GOPATH path and carry no go.mod. The
// go 1.21 module loads in module mode, as lint.sh runs the tool, so version
// gates read a real go directive.
func evalCorpora(t *testing.T) (string, []corpus) {
	t.Helper()

	// The go command reports module-mode paths with symlinks resolved, as in
	// macOS's /tmp, so the base they are made relative to must be resolved too.
	testdata, err := filepath.Abs(filepath.Join("analyzers", "testdata"))
	if err == nil {
		testdata, err = filepath.EvalSymlinks(testdata)
	}

	if err != nil {
		t.Fatalf("resolve corpus: %v", err)
	}

	src := filepath.Join(testdata, "src")

	return src, []corpus{
		{dir: src, env: []string{"GOPATH=" + testdata, "GO111MODULE=off", "GOFLAGS=", agentModeOn}},
		{dir: filepath.Join(testdata, "gomod121"), env: []string{"GOFLAGS=", agentModeOn}},
	}
}

// corpusFindings runs the tool over every corpus and returns what it reported
// as "file:line rule" strings, sorted.
func corpusFindings(t *testing.T, agents []string) []string {
	t.Helper()

	src, runs := evalCorpora(t)

	found := make([]string, 0, len(runs))
	for _, run := range runs {
		found = append(found, corpusRun(t, src, run, agents)...)
	}

	slices.Sort(found)

	return found
}

// corpusRun runs the tool once over one corpus with every analyzer, as lint.sh
// does. Plain output and the agent file carry no rule name, so the standard
// tier comes from -json, and each agent analyzer gets a run of its own with
// only it enabled. The solo runs together must write what the combined run's
// shared agent file holds.
func corpusRun(t *testing.T, src string, run corpus, agents []string) []string {
	t.Helper()

	combined := runTool(t, run.dir, run.env, "-json", "./...")
	if combined.code != exitClean {
		t.Fatalf("eval run in %s exit code %d; stderr:\n%s", run.dir, combined.code, combined.stderr)
	}

	found := jsonFindings(t, src, combined.stdout)

	solo := make([]string, 0, len(agents))

	for _, name := range agents {
		agentRun := runTool(t, run.dir, run.env, "-"+name, "./...")
		if agentRun.code != exitClean {
			t.Fatalf("eval run for %s in %s exit code %d; stderr:\n%s", name, run.dir, agentRun.code, agentRun.stderr)
		}

		found = append(found, agentFindings(t, src, name, agentRun.agentFile)...)
		solo = append(solo, fileLines(agentRun.agentFile)...)
	}

	missing, extra := multisetDiff(solo, fileLines(combined.agentFile))
	for _, line := range missing {
		t.Errorf("combined run in %s lost an agent finding: %s", run.dir, line)
	}

	for _, line := range extra {
		t.Errorf("combined run in %s wrote an agent finding no solo run did: %s", run.dir, line)
	}

	return found
}

// fileLines returns the lines of contents without their newlines.
func fileLines(contents string) []string {
	var lines []string
	for line := range strings.Lines(contents) {
		lines = append(lines, strings.TrimSuffix(line, "\n"))
	}

	return lines
}

// jsonDiagnostic is one finding in the multichecker's -json output.
type jsonDiagnostic struct {
	Posn    string `json:"posn"`
	Message string `json:"message"`
	rule    string
}

// jsonFindings reads -json output. The driver reports a package's non-test
// files again under its test variant, so a repeat of the same diagnostic
// counts once.
func jsonFindings(t *testing.T, src, output string) []string {
	t.Helper()

	var packages map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(output), &packages); err != nil {
		t.Fatalf("parse -json output: %v\n%s", err, output)
	}

	diags := make([]jsonDiagnostic, 0, len(packages))
	for pkg, byAnalyzer := range packages {
		diags = append(diags, packageDiagnostics(t, pkg, byAnalyzer)...)
	}

	seen := make(map[jsonDiagnostic]bool, len(diags))
	found := make([]string, 0, len(diags))

	for _, diag := range diags {
		if seen[diag] {
			continue
		}

		seen[diag] = true

		found = append(found, relativePosition(t, src, diag.Posn)+" "+diag.rule)
	}

	return found
}

// packageDiagnostics returns every analyzer's findings for one package,
// failing t when the driver reported an error there instead.
func packageDiagnostics(t *testing.T, pkg string, byAnalyzer map[string]json.RawMessage) []jsonDiagnostic {
	t.Helper()

	all := make([]jsonDiagnostic, 0, len(byAnalyzer))

	for rule, raw := range byAnalyzer {
		var diags []jsonDiagnostic
		if err := json.Unmarshal(raw, &diags); err != nil {
			t.Errorf("%s in %s did not run: %s", rule, pkg, raw)
		}

		for _, diag := range diags {
			diag.rule = rule
			all = append(all, diag)
		}
	}

	return all
}

// agentFindings reads the agent file of a run that enabled only rule.
func agentFindings(t *testing.T, src, rule, contents string) []string {
	t.Helper()

	var found []string

	for line := range strings.Lines(contents) {
		match := agentLine.FindStringSubmatch(line)
		if match == nil {
			t.Errorf("unreadable agent file line from %s: %q", rule, line)

			continue
		}

		found = append(found, relativePosition(t, src, match[1]+":"+match[2])+" "+rule)
	}

	return found
}

// relativePosition turns an absolute "path:line[:col]" into "path:line" with
// the path relative to src, so the eval file does not depend on the checkout.
func relativePosition(t *testing.T, src, position string) string {
	t.Helper()

	path, rest, _ := strings.Cut(position, ":")
	line, _, _ := strings.Cut(rest, ":")

	rel, err := filepath.Rel(src, path)
	if err != nil || !filepath.IsAbs(path) {
		t.Errorf("finding outside the corpus: %s", position)
	}

	return filepath.ToSlash(rel) + ":" + line
}

// splitFinding splits a "file:line rule" string into its parts.
func splitFinding(finding string) (string, int, string) {
	position, rule, _ := strings.Cut(finding, " ")
	path, line, _ := strings.Cut(position, ":")
	number, _ := strconv.Atoi(line)

	return path, number, rule
}

// readEval parses the eval file into its flag and quiet lines.
func readEval(t *testing.T) ([]string, []expectation) {
	t.Helper()

	var (
		flags []string
		quiet []expectation
	)

	data, err := os.ReadFile(evalFile)
	if err != nil {
		t.Fatalf("read eval file: %v", err)
	}

	for line := range strings.Lines(string(data)) {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}

		if len(fields) != 3 || (fields[0] != flagKind && fields[0] != quietKind) {
			t.Fatalf("eval file line %q is not \"flag|quiet file:line rule\"", strings.TrimSpace(line))
		}

		exp := expectation{kind: fields[0], position: fields[1], rule: fields[2]}
		if exp.kind == quietKind {
			quiet = append(quiet, exp)

			continue
		}

		flags = append(flags, exp.String())
	}

	slices.Sort(flags)

	return flags, quiet
}

// writeEval rewrites the eval file with found as its flag lines, keeping the
// hand-written quiet lines.
func writeEval(t *testing.T, found []string, quiet []expectation) {
	t.Helper()

	lines := make([]string, 0, len(found)+len(quiet)+1)
	lines = append(lines, strings.TrimSuffix(evalHeader, "\n"))

	for _, finding := range found {
		lines = append(lines, flagKind+" "+finding)
	}

	for _, exp := range quiet {
		lines = append(lines, quietKind+" "+exp.String())
	}

	if err := os.WriteFile(evalFile, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("write eval file: %v", err)
	}
}

// multisetDiff returns the entries of want missing from got and the entries
// of got missing from want. Both are sorted, and repeats count.
func multisetDiff(want, got []string) ([]string, []string) {
	var missing, extra []string

	counts := make(map[string]int, len(want))
	for _, entry := range want {
		counts[entry]++
	}

	for _, entry := range got {
		if counts[entry] > 0 {
			counts[entry]--

			continue
		}

		extra = append(extra, entry)
	}

	for _, entry := range want {
		if counts[entry] > 0 {
			counts[entry]--
			missing = append(missing, entry)
		}
	}

	return missing, extra
}

// TestEval runs the built tool over the analyzer fixtures and compares what
// it reports with the eval file, so a dependency update that loses a finding,
// adds one, or stops the tool from starting fails make test.
func TestEval(t *testing.T) {
	t.Parallel()

	_, agents := allListed(listAnalyzers(t))
	found := corpusFindings(t, slices.Sorted(maps.Keys(agents)))

	want, quiet := readEval(t)
	if *updateEval {
		writeEval(t, found, quiet)

		want = found
	}

	missing, extra := multisetDiff(want, found)
	for _, finding := range missing {
		t.Errorf("lost finding: %s", finding)
	}

	for _, finding := range extra {
		t.Errorf("new finding: %s", finding)
	}

	src, _ := evalCorpora(t)
	for _, exp := range quiet {
		checkQuiet(t, src, exp, found)
	}
}

// checkQuiet fails t when the rule reports anywhere in the near miss's scope:
// the top-level declaration holding the line, or the whole file outside every
// declaration. So a finding on a sibling line fails even after make
// eval-update records it. A line the file lacks fails too.
func checkQuiet(t *testing.T, src string, exp expectation, found []string) {
	t.Helper()

	path, line, _ := splitFinding(exp.String())

	first, last, ok := quietScope(t, filepath.Join(src, filepath.FromSlash(path)), line)
	if !ok {
		t.Errorf("near miss %s is not a line of a corpus file", exp)

		return
	}

	for _, finding := range found {
		if foundPath, foundLine, rule := splitFinding(finding); rule == exp.rule && foundPath == path &&
			foundLine >= first && foundLine <= last {
			t.Errorf("near miss flagged: %s (%s reports at line %d)", exp, rule, foundLine)
		}
	}
}

// quietScope returns the first and last line of the top-level declaration in
// file that holds line, its doc comment included, or the whole file when no
// declaration does. ok is false when the file is missing or lacks the line.
func quietScope(t *testing.T, file string, line int) (int, int, bool) {
	t.Helper()

	fset := token.NewFileSet()

	parsed, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return 0, 0, false
	}

	lastLine := fset.File(parsed.Pos()).LineCount()
	if line < 1 || line > lastLine {
		return 0, 0, false
	}

	for _, decl := range parsed.Decls {
		first := fset.PositionFor(declStart(decl), false).Line
		if last := fset.PositionFor(decl.End(), false).Line; line >= first && line <= last {
			return first, last, true
		}
	}

	return 1, lastLine, true
}

// declStart returns where decl begins, counting its doc comment.
func declStart(decl ast.Decl) token.Pos {
	switch typed := decl.(type) {
	case *ast.FuncDecl:
		if typed.Doc != nil {
			return typed.Doc.Pos()
		}
	case *ast.GenDecl:
		if typed.Doc != nil {
			return typed.Doc.Pos()
		}
	}

	return decl.Pos()
}

// TestEvalCoversEveryAnalyzer is the gate: every analyzer the built binary
// lists needs a must-flag line and a near-miss line in the eval file, and the
// eval file may not name a rule the binary no longer lists.
func TestEvalCoversEveryAnalyzer(t *testing.T) {
	t.Parallel()

	listed, _ := allListed(listAnalyzers(t))
	flags, quiet := readEval(t)

	flagged := make(map[string]bool, len(flags))

	for _, finding := range flags {
		_, rule, _ := strings.Cut(finding, " ")
		flagged[rule] = true
	}

	silent := make(map[string]bool, len(quiet))
	for _, exp := range quiet {
		silent[exp.rule] = true
	}

	for _, name := range listed {
		if !flagged[name] {
			t.Errorf("analyzer %s has no must-flag case in %s", name, evalFile)
		}

		if !silent[name] {
			t.Errorf("analyzer %s has no near-miss case in %s", name, evalFile)
		}
	}

	for rule := range mergedKeys(flagged, silent) {
		if !slices.Contains(listed, rule) {
			t.Errorf("%s names %s, which --list does not show", evalFile, rule)
		}
	}
}

// mergedKeys yields every key of left and right once.
func mergedKeys(left, right map[string]bool) func(func(string) bool) {
	return func(yield func(string) bool) {
		for key := range left {
			if !yield(key) {
				return
			}
		}

		for key := range right {
			if !left[key] && !yield(key) {
				return
			}
		}
	}
}
