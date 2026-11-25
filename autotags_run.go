package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// autoTagsRun is what every pass of [runAutoTagsPasses] shares: the streams
// merged output goes to and the binary each pass runs. PassArgs holds the
// arguments each pass forwards to that binary, and their package patterns
// also drive tag discovery. Every field must be set, and Stdout and Stderr
// must not be nil, because the passes write to them without a check.
type autoTagsRun struct {
	Stdout     io.Writer
	Stderr     io.Writer
	Executable string
	PassArgs   []string
}

// exitDiagnostics is the status the multichecker returns when it reported
// findings and nothing failed.
const exitDiagnostics = 3

// seenLines records the lines already written to each stream, kept apart so
// a stderr line that matches an earlier stdout line still prints.
type seenLines struct {
	stdout map[string]struct{}
	stderr map[string]struct{}
}

// runAutoTagsPasses re-executes run.Executable once per user-defined build
// tag under run.PassArgs' patterns, plus a pass with no -tags, deduping lines
// across passes. run must not be nil. A failed pass decides the exit code over
// passes with findings, so a broken build never reads as a lint failure. A
// subprocess per pass avoids the multichecker's global flags and os.Exit.
func runAutoTagsPasses(ctx context.Context, run *autoTagsRun) int {
	patterns := patternsFromArgs(run.PassArgs)

	tags, err := discoverBuildTags(patterns)
	if err != nil {
		_, _ = fmt.Fprintf(run.Stderr, "cairnlint: discover tags: %v\n", err)

		return 1
	}

	// Always run a default-build pass first so untagged code paths are
	// linted even when the project has no user tags.
	tagsets := append([]string{""}, tags...)

	seen := seenLines{stdout: map[string]struct{}{}, stderr: map[string]struct{}{}}

	var failure, findings int

	for _, tag := range tagsets {
		switch code := runSingleTagPass(ctx, run, tag, seen); {
		case code == exitDiagnostics:
			findings = code
		case code != 0 && failure == 0:
			failure = code
		}
	}

	if failure != 0 {
		return failure
	}

	return findings
}

// runSingleTagPass shells out to run.Executable with run.PassArgs,
// prepending -tags=<tag> when tag is non-empty. Output is captured and
// forwarded through seen so duplicate diagnostics across multiple passes
// collapse into one line in the merged output.
func runSingleTagPass(ctx context.Context, run *autoTagsRun, tag string, seen seenLines) int {
	label := "default build"
	childArgs := run.PassArgs

	if tag != "" {
		label = "-tags=" + tag
		childArgs = append([]string{"-tags=" + tag}, run.PassArgs...)
	}

	_, _ = fmt.Fprintf(run.Stderr, "cairnlint: pass %s\n", label)

	// run.Executable is obtained from os.Executable() at the top of main, so
	// the path is always this binary. gosec flags every exec.Command
	// called with a variable, but in this context the value is trusted.
	cmd := exec.CommandContext(ctx, run.Executable, childArgs...) // #nosec G204 G702 -- executable from os.Executable(); trusted self-path

	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()

	emitUnique(run.Stdout, outBuf.Bytes(), seen.stdout)
	emitUnique(run.Stderr, errBuf.Bytes(), seen.stderr)

	if exitErr, ok := errors.AsType[*exec.ExitError](runErr); ok {
		return exitErr.ExitCode()
	}

	if runErr != nil {
		_, _ = fmt.Fprintf(run.Stderr, "cairnlint: %s: %v\n", label, runErr)

		return 1
	}

	return 0
}

// emitUnique writes each non-empty line from data to w the first time it
// appears. seen is shared across passes so the same diagnostic from two tag
// configurations does not print twice, and each stream gets its own, so a
// line only counts as a repeat within its stream.
func emitUnique(dst io.Writer, data []byte, seen map[string]struct{}) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Diagnostic lines are short, but a defensive buffer cap keeps us
	// from truncating pathological panic traces.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if _, dup := seen[line]; dup {
			continue
		}

		seen[line] = struct{}{}

		_, _ = fmt.Fprintln(dst, line)
	}
}

// patternsFromArgs returns the positional package pattern arguments from
// args, dropping any -flag or --flag tokens. Every cairnlint flag the
// multichecker accepts takes its value in =form (e.g. -tags=foo), so a
// bare leading-dash token is always a flag rather than a pattern.
func patternsFromArgs(args []string) []string {
	patterns := make([]string, 0, len(args))

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}

		patterns = append(patterns, arg)
	}

	return patterns
}
