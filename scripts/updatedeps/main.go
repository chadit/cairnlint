// Command updatedeps moves every requirement in go.mod, and the go directive,
// to the newest release that has been public for at least -min-age-days days.
// A hijacked release is usually found and pulled within days, so nothing
// younger than the wait enters the tree. `make update-deps` runs it from the
// repository root.
//
// Each move runs `go get` and `go mod tidy`, then checks every version the
// move changed. When minimal version selection raised another module or the
// Go version to one inside the wait, or to one with no known publish time,
// go.mod and go.sum are restored and the item stays where it was.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

// goItem names the go directive among the module paths of a go.mod state.
const goItem = "go"

// Go releases are published as versions of this module, one per platform,
// within minutes of each other, so one platform's listing dates them all.
const (
	toolchainModule = "golang.org/toolchain"
	toolchainPrefix = "v0.0.1-go"
	toolchainSuffix = ".linux-amd64"
)

// modFileMode only applies if a restore has to recreate a file tidy removed.
const modFileMode os.FileMode = 0o600

// release is one version an item can move to: the go list query that dates
// it and the go get argument that selects it.
type release struct {
	query  string
	getArg string
}

type updater struct {
	log    *slog.Logger
	cutoff time.Time
}

func main() {
	minAgeDays := flag.Int("min-age-days", 0, "skip versions published fewer than this many days ago")

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if *minAgeDays < 1 {
		logger.Error("-min-age-days must be at least 1", "got", *minAgeDays)
		os.Exit(1)
	}

	upd := updater{
		log:    logger,
		cutoff: time.Now().AddDate(0, 0, -*minAgeDays),
	}
	if err := upd.run(context.Background()); err != nil {
		logger.Error("update failed", "error", err)
		os.Exit(1)
	}
}

// run visits each go.mod item once, re-reading go.mod before every step so a
// module that an earlier move added or raised is judged from where it now is.
func (u *updater) run(ctx context.Context) error {
	visited := map[string]bool{}

	for {
		state, err := readState(ctx)
		if err != nil {
			return err
		}

		name, ok := nextItem(state, visited)
		if !ok {
			return nil
		}

		visited[name] = true

		if err := u.updateItem(ctx, name, state[name]); err != nil {
			return err
		}
	}
}

func nextItem(state map[string]string, visited map[string]bool) (string, bool) {
	for _, name := range slices.Sorted(maps.Keys(state)) {
		if !visited[name] {
			return name, true
		}
	}

	return "", false
}

func (u *updater) updateItem(ctx context.Context, name, current string) error {
	releases, err := listReleases(ctx, name)
	if err != nil {
		return err
	}

	currentSemver := semverOf(name, current)

	newer := newerReleases(currentSemver, slices.Collect(maps.Keys(releases)))
	if len(newer) == 0 {
		return nil
	}

	queries := make([]string, 0, len(newer))
	for _, v := range newer {
		queries = append(queries, releases[v].query)
	}

	times, err := publishTimes(ctx, queries)
	if err != nil {
		return err
	}

	published := make(map[string]time.Time, len(releases))
	for v, rel := range releases {
		published[v] = times[rel.query]
	}

	pick, err := pickVersion(currentSemver, published, u.cutoff)
	if err != nil {
		u.log.Warn("leaving item where it is", "item", name, "version", current, "reason", err)

		return nil
	}

	if pick != newer[0] {
		u.log.Info("newest version is inside the wait", "item", name, "newest", newer[0],
			"published", published[newer[0]].Format(time.RFC3339), "moving_to", pick)
	}

	if pick == "" {
		return nil
	}

	return u.apply(ctx, name, releases[pick].getArg)
}

// apply moves one item and keeps the move only when every other version it
// changed is old enough.
func (u *updater) apply(ctx context.Context, name, getArg string) error {
	before, err := readState(ctx)
	if err != nil {
		return err
	}

	saved, err := saveModFiles()
	if err != nil {
		return err
	}

	_, getErr := runGo(ctx, "get", getArg)
	if getErr == nil {
		_, getErr = runGo(ctx, "mod", "tidy")
	}

	if getErr != nil {
		return fmt.Errorf("move %s: %w", getArg, errors.Join(getErr, saved.restore()))
	}

	young, err := u.youngChanges(ctx, before, name)
	if err != nil {
		return fmt.Errorf("check %s: %w", getArg, errors.Join(err, saved.restore()))
	}

	if len(young) > 0 {
		u.log.Warn("leaving item where it is: the move pulls in versions inside the wait or with no publish time",
			"item", name, "wanted", getArg, "pulls_in", strings.Join(young, ", "))

		return saved.restore()
	}

	u.log.Info("moved", "item", name, "to", getArg)

	return nil
}

// youngChanges lists every version other than the moved item's own that rose
// since before and is inside the wait or has no known publish time.
func (u *updater) youngChanges(ctx context.Context, before map[string]string, moved string) ([]string, error) {
	after, err := readState(ctx)
	if err != nil {
		return nil, err
	}

	var queries []string

	for name, version := range after {
		old, existed := before[name]
		if name == moved || (existed && semver.Compare(semverOf(name, version), semverOf(name, old)) <= 0) {
			continue
		}

		queries = append(queries, queryOf(name, version))
	}

	if len(queries) == 0 {
		return nil, nil
	}

	times, err := publishTimes(ctx, queries)
	if err != nil {
		return nil, err
	}

	var young []string

	for _, query := range slices.Sorted(slices.Values(queries)) {
		if published := times[query]; published.IsZero() || published.After(u.cutoff) {
			young = append(young, query)
		}
	}

	return young, nil
}

// pickVersion returns the newest version in published that is newer than
// current and was public at cutoff, or "" when none is. A zero time means the
// publish time is unknown; meeting one before an old-enough version returns
// errNoPublishTime, because that version might be the one to skip or take.
func pickVersion(current string, published map[string]time.Time, cutoff time.Time) (string, error) {
	for _, version := range newerReleases(current, slices.Collect(maps.Keys(published))) {
		at := published[version]
		if at.IsZero() {
			return "", fmt.Errorf("%w: %s", errNoPublishTime, version)
		}

		if !at.After(cutoff) {
			return version, nil
		}
	}

	return "", nil
}

// newerReleases returns the versions newer than current, newest first. Like
// go get's "latest", it leaves pre-releases out whenever versions holds any
// release.
func newerReleases(current string, versions []string) []string {
	hasRelease := slices.ContainsFunc(versions, isRelease)

	var newer []string

	for _, version := range versions {
		if semver.Compare(version, current) > 0 && (!hasRelease || isRelease(version)) {
			newer = append(newer, version)
		}
	}

	slices.SortFunc(newer, func(a, b string) int { return semver.Compare(b, a) })

	return newer
}

func isRelease(version string) bool {
	return semver.Prerelease(version) == ""
}

// goToSemver turns a Go version such as 1.27.4, 1.28rc1, or 1.20 into a
// semantic version so it orders with the same rules as module versions.
func goToSemver(goVersion string) string {
	base, pre := goVersion, ""
	if i := strings.IndexAny(goVersion, "abcdefghijklmnopqrstuvwxyz"); i >= 0 {
		base, pre = goVersion[:i], "-"+goVersion[i:]
	}

	if strings.Count(base, ".") == 1 {
		base += ".0"
	}

	return "v" + base + pre
}

func semverOf(name, version string) string {
	if name == goItem {
		return goToSemver(version)
	}

	return version
}

func queryOf(name, version string) string {
	if name == goItem {
		return toolchainModule + "@" + toolchainPrefix + version + toolchainSuffix
	}

	return name + "@" + version
}

// listReleases returns every version the item can move to, keyed by semver.
func listReleases(ctx context.Context, name string) (map[string]release, error) {
	if name != goItem {
		versions, err := listVersions(ctx, name)
		if err != nil {
			return nil, err
		}

		releases := make(map[string]release, len(versions))
		for _, v := range versions {
			releases[v] = release{query: name + "@" + v, getArg: name + "@" + v}
		}

		return releases, nil
	}

	versions, err := listVersions(ctx, toolchainModule)
	if err != nil {
		return nil, err
	}

	releases := make(map[string]release, len(versions))

	for _, v := range versions {
		goVersion, ok := strings.CutPrefix(v, toolchainPrefix)
		goVersion, onPlatform := strings.CutSuffix(goVersion, toolchainSuffix)

		if sv := goToSemver(goVersion); ok && onPlatform && semver.IsValid(sv) {
			releases[sv] = release{query: toolchainModule + "@" + v, getArg: "go@" + goVersion}
		}
	}

	return releases, nil
}

func listVersions(ctx context.Context, modulePath string) ([]string, error) {
	out, err := runGo(ctx, "list", "-m", "-json", "-versions", modulePath)
	if err != nil {
		return nil, err
	}

	// encoding/json matches keys without regard to case, so this file's
	// lowercase tags read the capitalized fields go's -json output writes.
	var listed struct {
		Versions []string `json:"versions"`
	}
	if err := json.Unmarshal(out, &listed); err != nil {
		return nil, fmt.Errorf("decode versions of %s: %w", modulePath, err)
	}

	return listed.Versions, nil
}

// publishTimes dates each path@version query. A query the proxy cannot
// answer is left out of the map, so its time reads as zero.
func publishTimes(ctx context.Context, queries []string) (map[string]time.Time, error) {
	out, err := runGo(ctx, append([]string{"list", "-m", "-e", "-json"}, queries...)...)
	if err != nil {
		return nil, err
	}

	times := make(map[string]time.Time, len(queries))

	for dec := json.NewDecoder(bytes.NewReader(out)); ; {
		var mod struct {
			Time    *time.Time `json:"time"`
			Path    string     `json:"path"`
			Version string     `json:"version"`
		}

		err := dec.Decode(&mod)
		if errors.Is(err, io.EOF) {
			return times, nil
		}

		if err != nil {
			return nil, fmt.Errorf("decode publish times: %w", err)
		}

		if mod.Time != nil {
			times[mod.Path+"@"+mod.Version] = *mod.Time
		}
	}
}

// readState maps each go.mod requirement to its version, plus the go
// directive under goItem.
func readState(ctx context.Context) (map[string]string, error) {
	out, err := runGo(ctx, "mod", "edit", "-json")
	if err != nil {
		return nil, err
	}

	var mod struct {
		Go      string `json:"go"`
		Require []struct {
			Path    string `json:"path"`
			Version string `json:"version"`
		} `json:"require"`
	}
	if err := json.Unmarshal(out, &mod); err != nil {
		return nil, fmt.Errorf("decode go.mod: %w", err)
	}

	state := map[string]string{goItem: mod.Go}
	for _, req := range mod.Require {
		state[req.Path] = req.Version
	}

	return state, nil
}

func runGo(ctx context.Context, args ...string) ([]byte, error) {
	var stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, "go", args...) // #nosec G204 -- always the go tool; callers pass go subcommands and module queries
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go %s: %w: %s", strings.Join(args, " "), err, bytes.TrimSpace(stderr.Bytes()))
	}

	return out, nil
}

// modFiles holds go.mod and go.sum as they were before a move, so a rejected
// move can be undone byte for byte.
type modFiles map[string][]byte

func saveModFiles() (modFiles, error) {
	saved := modFiles{}

	for _, name := range []string{"go.mod", "go.sum"} {
		content, err := os.ReadFile(name) // #nosec G304 -- name is go.mod or go.sum in the working directory
		if errors.Is(err, os.ErrNotExist) {
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("save %s: %w", name, err)
		}

		saved[name] = content
	}

	return saved, nil
}

func (m modFiles) restore() error {
	if _, hadSum := m["go.sum"]; !hadSum {
		if err := os.Remove("go.sum"); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("restore go.sum: %w", err)
		}
	}

	for name, content := range m {
		if err := os.WriteFile(name, content, modFileMode); err != nil {
			return fmt.Errorf("restore %s: %w", name, err)
		}
	}

	return nil
}
