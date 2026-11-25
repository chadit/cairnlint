package main_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/mod/modfile"
)

// The fixture module every case upgrades, and the wait each run asks for.
const (
	modulePath = "example.com/m"
	minAgeDays = "3"
)

// Versions several cases start from or expect.
const (
	baseVersion  = "v1.0.0"
	laterVersion = "v1.2.0"
	baseGo       = "1.21.4"
)

// binaryPath is the updatedeps program TestMain builds, so each case runs it
// the way make update-deps does rather than calling into package main.
var binaryPath string //nolint:gochecknoglobals // written once by TestMain before any test runs

func TestMain(m *testing.M) {
	os.Exit(runWithBinary(m))
}

func runWithBinary(m *testing.M) int {
	dir, err := os.MkdirTemp("", "updatedeps-bin-")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "E: create binary dir: %v\n", err)

		return 1
	}

	defer func() { _ = os.RemoveAll(dir) }()

	binaryPath = filepath.Join(dir, "updatedeps")

	build := exec.CommandContext(context.Background(), "go", "build", "-o", binaryPath, ".")
	build.Stderr = os.Stderr

	if err := build.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "E: build updatedeps: %v\n", err)

		return 1
	}

	return m.Run()
}

// fixtureVersion is one version the fixture proxy serves. A zero published
// time serves the version with no publish time.
type fixtureVersion struct {
	published time.Time
	version   string
}

// Publish times either side of the -min-age-days wait. They are days away
// from it, so the moment the program reads the clock cannot change a case.
func oldRelease() time.Time   { return time.Now().AddDate(0, 0, -30) }
func youngRelease() time.Time { return time.Now().Add(-24 * time.Hour) }

func at(published time.Time, versions ...string) []fixtureVersion {
	out := make([]fixtureVersion, 0, len(versions))
	for _, version := range versions {
		out = append(out, fixtureVersion{version: version, published: published})
	}

	return out
}

// writeFile writes contents at path, creating its directory.
func writeFile(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create %s: %v", filepath.Dir(path), err)
	}

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// infoJSON is the proxy's .info answer for one version.
func infoJSON(release fixtureVersion) string {
	if release.published.IsZero() {
		return fmt.Sprintf(`{"Version":%q}`, release.version)
	}

	return fmt.Sprintf(`{"Version":%q,"Time":%q}`, release.version, release.published.UTC().Format(time.RFC3339))
}

// writeModuleVersions serves modulePath at each version, with the zip go get
// and go mod tidy download.
func writeModuleVersions(t *testing.T, proxy string, versions []fixtureVersion) {
	t.Helper()

	dir := filepath.Join(proxy, modulePath, "@v")
	goMod := "module " + modulePath + "\n\ngo 1.21\n"

	list := make([]string, 0, len(versions))

	for _, release := range versions {
		list = append(list, release.version)
		writeFile(t, filepath.Join(dir, release.version+".info"), infoJSON(release))
		writeFile(t, filepath.Join(dir, release.version+".mod"), goMod)
		writeFile(t, filepath.Join(dir, release.version+".zip"), moduleZip(t, release.version, goMod))
	}

	writeFile(t, filepath.Join(dir, "list"), strings.Join(list, "\n")+"\n")
}

// moduleZip returns a module zip holding go.mod and one source file.
func moduleZip(t *testing.T, version, goMod string) string {
	t.Helper()

	var buf bytes.Buffer

	archive := zip.NewWriter(&buf)
	prefix := modulePath + "@" + version + "/"

	for name, contents := range map[string]string{"go.mod": goMod, "m.go": "package m\n\n// V is a value.\nconst V = 1\n"} {
		entry, err := archive.Create(prefix + name)
		if err != nil {
			t.Fatalf("zip %s: %v", name, err)
		}

		if _, err := entry.Write([]byte(contents)); err != nil {
			t.Fatalf("zip %s: %v", name, err)
		}
	}

	if err := archive.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	return buf.String()
}

// writeGoReleases serves each Go version, such as 1.21.10 or 1.22rc1, the way
// the proxy lists Go releases: as linux-amd64 versions of golang.org/toolchain.
func writeGoReleases(t *testing.T, proxy string, releases []fixtureVersion) {
	t.Helper()

	dir := filepath.Join(proxy, "golang.org", "toolchain", "@v")

	list := make([]string, 0, len(releases))

	for _, release := range releases {
		version := "v0.0.1-go" + release.version + ".linux-amd64"
		list = append(list, version)
		writeFile(t, filepath.Join(dir, version+".info"), infoJSON(fixtureVersion{version: version, published: release.published}))
		writeFile(t, filepath.Join(dir, version+".mod"), "module golang.org/toolchain\n")
	}

	writeFile(t, filepath.Join(dir, "list"), strings.Join(list, "\n")+"\n")
}

// upgradeCase is a go.mod in, the versions the proxy offers, and the go.mod
// the program must leave.
type upgradeCase struct {
	name        string
	goIn        string
	requireIn   string
	wantGo      string
	wantRequire string
	wantStderr  string
	versions    []fixtureVersion
	goReleases  []fixtureVersion
}

// updateResult is what one run of the program left for the user to see.
type updateResult struct {
	stderr  string
	goLine  string
	require string
	code    int
}

// runUpdate writes the case's proxy and module, runs the program from the
// module root with -min-age-days, and reads back the go.mod it left. The proxy
// is a file:// URL with no fallback, so nothing reaches the network, and
// GOTOOLCHAIN=local keeps go from fetching a toolchain.
func runUpdate(t *testing.T, testCase *upgradeCase) updateResult {
	t.Helper()

	root := t.TempDir()
	proxy := filepath.Join(root, "proxy")
	app := filepath.Join(root, "app")

	writeModuleVersions(t, proxy, testCase.versions)
	writeGoReleases(t, proxy, testCase.goReleases)
	writeFile(t, filepath.Join(app, "go.mod"),
		"module example.com/app\n\ngo "+testCase.goIn+"\n\nrequire "+modulePath+" "+testCase.requireIn+"\n")
	writeFile(t, filepath.Join(app, "app.go"), "package app\n\nimport \"example.com/m\"\n\n// V forwards.\nconst V = m.V\n")

	env := append(os.Environ(),
		"GOPROXY=file://"+filepath.ToSlash(proxy), "GOSUMDB=off", "GOFLAGS=-modcacherw",
		"GOMODCACHE="+filepath.Join(root, "modcache"), "GOTOOLCHAIN=local", "GOWORK=off",
		"GONOPROXY=", "GOPRIVATE=", "GONOSUMDB=", "GOINSECURE=")

	// A checked-in module has a go.sum, which the program's go commands need.
	tidy := exec.CommandContext(t.Context(), "go", "mod", "tidy")
	tidy.Dir, tidy.Env = app, env

	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("prepare go.sum: %v\n%s", err, out)
	}

	cmd := exec.CommandContext(t.Context(), binaryPath, "-min-age-days", minAgeDays)
	cmd.Dir, cmd.Env = app, env

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	var code int

	err := cmd.Run()

	exitErr, exited := errors.AsType[*exec.ExitError](err)
	if err != nil && !exited {
		t.Fatalf("start updatedeps: %v", err)
	}

	if exited {
		code = exitErr.ExitCode()
	}

	return readResult(t, filepath.Join(app, "go.mod"), stderr.String(), code)
}

// readResult parses the go.mod the run left.
func readResult(t *testing.T, path, stderr string, code int) updateResult {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	parsed, err := modfile.Parse(path, data, nil)
	if err != nil {
		t.Fatalf("parse go.mod: %v\n%s", err, data)
	}

	result := updateResult{stderr: stderr, code: code, goLine: parsed.Go.Version}

	for _, req := range parsed.Require {
		if req.Mod.Path == modulePath {
			result.require = req.Mod.Version
		}
	}

	return result
}

// checkUpgrade runs one case and compares the go.mod it left.
func checkUpgrade(t *testing.T, testCase *upgradeCase) {
	t.Helper()

	got := runUpdate(t, testCase)
	if got.code != 0 {
		t.Fatalf("exit code %d, want 0; stderr:\n%s", got.code, got.stderr)
	}

	if got.require != testCase.wantRequire || got.goLine != testCase.wantGo {
		t.Errorf("go.mod has go %s and %s %s, want go %s and %s; stderr:\n%s",
			got.goLine, modulePath, got.require, testCase.wantGo, testCase.wantRequire, got.stderr)
	}

	if !strings.Contains(got.stderr, testCase.wantStderr) {
		t.Errorf("stderr lacks %q:\n%s", testCase.wantStderr, got.stderr)
	}
}

func TestModuleUpgrades(t *testing.T) {
	t.Parallel()

	old, young := oldRelease(), youngRelease()

	cases := []upgradeCase{
		{
			name:      "newest old-enough version when newer ones are too young",
			requireIn: "v0.50.0", wantRequire: "v0.52.0", wantStderr: "v0.53.0",
			versions: append(at(old, "v0.50.0", "v0.51.0", "v0.52.0"), at(young, "v0.53.0")...),
		},
		{
			name:      "nothing newer is old enough",
			requireIn: baseVersion, wantRequire: baseVersion,
			versions: append(at(old, baseVersion), at(young, "v1.1.0", laterVersion)...),
		},
		{
			name:      "already at the newest version",
			requireIn: laterVersion, wantRequire: laterVersion,
			versions: at(old, "v1.1.0", laterVersion),
		},
		{
			name:      "missing time on a version that outranks the pick",
			requireIn: baseVersion, wantRequire: baseVersion, wantStderr: laterVersion,
			versions: append(at(old, baseVersion, "v1.1.0"), at(time.Time{}, laterVersion)...),
		},
		{
			name:      "missing time below the pick does not matter",
			requireIn: baseVersion, wantRequire: laterVersion,
			versions: append(at(old, baseVersion, laterVersion), at(time.Time{}, "v1.1.0")...),
		},
		{
			name:      "pre-release is not picked over a release",
			requireIn: baseVersion, wantRequire: "v1.1.0",
			versions: at(old, baseVersion, "v1.1.0", "v1.2.0-rc.1"),
		},
		{
			name:      "pre-release with no release newer than current stays put",
			requireIn: baseVersion, wantRequire: baseVersion,
			versions: at(old, baseVersion, "v1.1.0-rc.1"),
		},
		{
			name:      "pre-release is picked when the module has no releases",
			requireIn: "v0.1.0-alpha", wantRequire: "v0.1.0-beta",
			versions: at(old, "v0.1.0-alpha", "v0.1.0-beta"),
		},
	}

	for _, testCase := range cases {
		testCase.goIn, testCase.wantGo = baseGo, baseGo

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			checkUpgrade(t, &testCase)
		})
	}
}

// TestGoDirectiveUpgrades checks how Go versions order against each other:
// patch numbers compare as numbers, a two-part version sits above the patch
// releases before it, and release candidates and betas order as pre-releases.
// Every target stays at or below the toolchain running the test, since
// GOTOOLCHAIN=local cannot move go.mod past it.
func TestGoDirectiveUpgrades(t *testing.T) {
	t.Parallel()

	old := oldRelease()

	cases := []upgradeCase{
		{
			name: "patch releases order as numbers and a release candidate is skipped",
			goIn: baseGo, wantGo: "1.21.10",
			goReleases: at(old, "1.21.9", "1.21.10", "1.22rc1"),
		},
		{
			name: "a two-part version orders above older patch releases",
			goIn: baseGo, wantGo: "1.22",
			goReleases: at(old, "1.21.13", "1.22"),
		},
		{
			name: "a release candidate orders above a beta when there are only pre-releases",
			goIn: baseGo, wantGo: "1.22rc1",
			goReleases: at(old, "1.22beta2", "1.22rc1"),
		},
		{
			name: "a three-part .0 directive is the starting point",
			goIn: "1.21.0", wantGo: "1.21.1",
			goReleases: at(old, "1.21.0", "1.21.1"),
		},
		{
			name: "a newer Go release inside the wait is left alone",
			goIn: baseGo, wantGo: baseGo,
			goReleases: at(youngRelease(), "1.21.5"),
		},
	}

	for _, testCase := range cases {
		testCase.requireIn, testCase.wantRequire = baseVersion, baseVersion
		testCase.versions = at(old, baseVersion)

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			checkUpgrade(t, &testCase)
		})
	}
}
