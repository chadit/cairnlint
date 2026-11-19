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

// AutoTagsRun is what every pass of [RunAutoTagsPasses] shares: the streams
// merged output goes to and the binary each pass runs. PassArgs holds the
// arguments each pass forwards to that binary, and their package patterns
// also drive tag discovery. Every field must be set, and Stdout and Stderr
// must not be nil, because the passes write to them without a check.
type AutoTagsRun struct {
	Stdout     io.Writer
	Stderr     io.Writer
	Executable string
	PassArgs   []string
}

// RunAutoTagsPasses re-executes run.Executable once per user-defined build
// tag found under run.PassArgs' patterns, plus one pass with no -tags. run
// must not be nil. It dedupes output lines across passes and returns the
// worst exit code. A subprocess per pass avoids the global flags and os.Exit
// inside the multichecker driver.
func RunAutoTagsPasses(ctx context.Context, run *AutoTagsRun) int {
	patterns := patternsFromArgs(run.PassArgs)

	tags, err := DiscoverBuildTags(patterns)
	if err != nil {
		_, _ = fmt.Fprintf(run.Stderr, "cairnlint: discover tags: %v\n", err)

		return 1
	}

	// Always run a default-build pass first so untagged code paths are
	// linted even when the project has no user tags.
	tagsets := append([]string{""}, tags...)

	seen := make(map[string]struct{}, len(tagsets))

	var worst int

	for _, tag := range tagsets {
		code := runSingleTagPass(ctx, run, tag, seen)
		if code > worst {
			worst = code
		}
	}

	return worst
}

// runSingleTagPass shells out to run.Executable with run.PassArgs,
// prepending -tags=<tag> when tag is non-empty. Output is captured and
// forwarded through seen so duplicate diagnostics across multiple passes
// collapse into one line in the merged output.
func runSingleTagPass(ctx context.Context, run *AutoTagsRun, tag string, seen map[string]struct{}) int {
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

	emitUnique(run.Stdout, outBuf.Bytes(), seen)
	emitUnique(run.Stderr, errBuf.Bytes(), seen)

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
