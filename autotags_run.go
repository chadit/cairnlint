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

// RunAutoTagsPasses re-executes the cairnlint binary once per user-defined
// build tag discovered under passArgs' patterns, plus a default-build
// pass with no -tags. Output lines are deduped across passes so callers
// see every unique diagnostic exactly once regardless of which build
// configuration surfaced it. Returns the worst exit code observed so the
// overall invocation fails if any pass reports issues.
//
// Using a subprocess rather than calling multichecker.Main in-process
// avoids the global-flag and os.Exit coupling inside the multichecker
// driver; each pass gets a clean process environment.
func RunAutoTagsPasses(ctx context.Context, stdout, stderr io.Writer, executable string, passArgs []string) int {
	patterns := patternsFromArgs(passArgs)

	tags, err := DiscoverBuildTags(patterns)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "cairnlint: discover tags: %v\n", err)

		return 1
	}

	// Always run a default-build pass first so untagged code paths are
	// linted even when the project has no user tags.
	tagsets := append([]string{""}, tags...)

	seen := make(map[string]struct{}, len(tagsets))

	var worst int

	for _, tag := range tagsets {
		code := runSingleTagPass(ctx, stdout, stderr, executable, tag, passArgs, seen)
		if code > worst {
			worst = code
		}
	}

	return worst
}

// runSingleTagPass shells out to executable with the supplied passArgs,
// prepending -tags=<tag> when tag is non-empty. Output is captured and
// forwarded through seen so duplicate diagnostics across multiple passes
// collapse into one line in the merged output.
func runSingleTagPass(ctx context.Context, stdout, stderr io.Writer, executable, tag string, passArgs []string, seen map[string]struct{}) int {
	label := "default build"
	childArgs := passArgs

	if tag != "" {
		label = "-tags=" + tag
		childArgs = append([]string{"-tags=" + tag}, passArgs...)
	}

	_, _ = fmt.Fprintf(stderr, "cairnlint: pass %s\n", label)

	// executable is obtained from os.Executable() at the top of main, so
	// the path is always this binary. gosec flags every exec.Command
	// called with a variable, but in this context the value is trusted.
	cmd := exec.CommandContext(ctx, executable, childArgs...) // #nosec G204 G702 -- executable from os.Executable(); trusted self-path

	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()

	emitUnique(stdout, outBuf.Bytes(), seen)
	emitUnique(stderr, errBuf.Bytes(), seen)

	if exitErr, ok := errors.AsType[*exec.ExitError](runErr); ok {
		return exitErr.ExitCode()
	}

	if runErr != nil {
		_, _ = fmt.Fprintf(stderr, "cairnlint: %s: %v\n", label, runErr)

		return 1
	}

	return 0
}

// emitUnique writes each non-empty line from data to w the first time it
// appears. A single seen map is shared across passes so the same
// diagnostic from two tag configurations does not print twice.
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
