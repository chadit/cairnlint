package analyzers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/tools/go/analysis"
)

// agentFileMode keeps findings readable only by the user who ran the linter,
// since the file sits in a shared temp directory.
const agentFileMode = 0o600

// Agent files are named agentFilePrefix + PID + agentFileSuffix.
const (
	agentFilePrefix = "cairnlint-agent-"
	agentFileSuffix = ".txt"
)

// staleAgentFileAge is how old an agent file must be before a later run
// removes it. The caller reads its file right after the run exits, so a day
// leaves it plenty of time while keeping the temp directory from filling up.
const staleAgentFileAge = 24 * time.Hour

// SweepStaleAgentFiles removes agent files in the temp directory whose
// modification time is more than staleAgentFileAge ago. The tool cannot
// delete its own file, because the caller reads it after the run, so each
// agent-mode run clears what earlier runs left. Entries it cannot read or
// remove, such as another user's file in a shared /tmp, are skipped silently.
func SweepStaleAgentFiles() {
	dir := os.TempDir()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-staleAgentFileAge)

	for _, entry := range entries {
		if isStaleAgentFile(entry, cutoff) {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}

// isStaleAgentFile reports whether entry is a regular file named like an agent
// file and last modified before cutoff. ReadDir entries describe the entry
// itself, so a symlink is never taken for the file it points at.
func isStaleAgentFile(entry os.DirEntry, cutoff time.Time) bool {
	if !isAgentFileName(entry.Name()) || !entry.Type().IsRegular() {
		return false
	}

	info, err := entry.Info()

	return err == nil && info.ModTime().Before(cutoff)
}

// isAgentFileName reports whether name is agentFilePrefix, one or more
// digits, and agentFileSuffix.
func isAgentFileName(name string) bool {
	pidAndSuffix, hasPrefix := strings.CutPrefix(name, agentFilePrefix)
	pid, hasSuffix := strings.CutSuffix(pidAndSuffix, agentFileSuffix)

	return hasPrefix && hasSuffix && pid != "" && strings.Trim(pid, "0123456789") == ""
}

// agentFileWriter appends agent-tier diagnostics to a per-process temp file,
// keeping them out of the stdout stream humans and CI read. It holds no open
// handle: the framework has no end-of-run hook to close one, and opening at
// construction would leave an empty file after a clean run. Each line opens,
// writes, and closes the file under mu instead.
type agentFileWriter struct {
	written     map[agentFinding]bool
	path        string
	mu          sync.Mutex
	summaryOnce sync.Once
	opened      bool
}

// agentFinding keys a written line the way the driver's printer keys a
// diagnostic, by analyzer and printed text. The driver analyzes a package's
// non-test files again in its test variant and prints the repeat once; this
// writer bypasses that printer, so it skips the repeat itself.
type agentFinding struct {
	analyzer string
	line     string
}

func newAgentFileWriter() *agentFileWriter {
	return &agentFileWriter{
		path:    filepath.Join(os.TempDir(), fmt.Sprintf("%s%d%s", agentFilePrefix, os.Getpid(), agentFileSuffix)),
		written: make(map[agentFinding]bool),
	}
}

// writeLine appends finding's line to the agent file, creating it on the
// first call, and does nothing for a finding already written. The first open
// also truncates, so a file an earlier process with the same PID left behind
// does not carry its findings into this run.
func (writer *agentFileWriter) writeLine(finding agentFinding) error {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if writer.written[finding] {
		return nil
	}

	// The path is predictable, so agentOpenFlags refuse what someone else may
	// have planted there, and a refused finding falls back to stderr instead.
	flags := os.O_APPEND | os.O_CREATE | os.O_WRONLY | agentOpenFlags
	if !writer.opened {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(writer.path, flags, agentFileMode)
	if err != nil {
		return fmt.Errorf("open agent file: %w", err)
	}

	// A FIFO with a reader or a device still opens, so only a regular file
	// is written to.
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat agent file %s: %w", writer.path, errors.Join(err, file.Close()))
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("agent file %s: %w", writer.path, errors.Join(errNotRegularFile, file.Close()))
	}

	// OpenFile's mode applies only when it creates the file, so a leftover
	// file from a reused PID keeps its old mode until this resets it.
	if !writer.opened {
		if err := file.Chmod(agentFileMode); err != nil {
			return fmt.Errorf("restrict agent file %s: %w", writer.path, errors.Join(err, file.Close()))
		}
	}

	writer.opened = true

	_, writeErr := fmt.Fprintln(file, finding.line)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return fmt.Errorf("write agent file %s: %w", writer.path, err)
	}

	writer.written[finding] = true

	return nil
}

// wrapAgentFileOutput redirects agent diagnostics to a temp file and prints
// one stderr line on the first finding so the caller knows where to look. A
// diagnostic the file cannot take falls back to pass.Report, which the driver
// prints to stderr. It replaces pass.Report without forwarding, so it must be
// the outermost wrapper; enter through WrapForAgentFile.
func wrapAgentFileOutput(all []*analysis.Analyzer) []*analysis.Analyzer {
	writer := newAgentFileWriter()

	wrapped := make([]*analysis.Analyzer, len(all))
	for idx, orig := range all {
		wrapped[idx] = wrapAgentOut(orig, writer)
	}

	return wrapped
}

// WrapForAgentFile routes diagnostics from all to the agent file after the
// suppression and generated-file filters run. The file writer never calls the
// Report it replaces, so it must wrap outermost: the other way round, neither
// filter sees an agent diagnostic and //nolint has no effect on the agent tier.
func WrapForAgentFile(all []*analysis.Analyzer) []*analysis.Analyzer {
	return wrapAgentFileOutput(WrapWithNolint(WrapSkipGenerated(all)))
}

func wrapAgentOut(orig *analysis.Analyzer, writer *agentFileWriter) *analysis.Analyzer {
	originalRun := orig.Run

	wrapped := *orig
	wrapped.Run = func(pass *analysis.Pass) (any, error) {
		filteredPass := *pass

		filteredPass.Report = func(diag analysis.Diagnostic) {
			pos := pass.Fset.Position(diag.Pos)
			if err := writer.writeLine(agentFinding{analyzer: orig.Name, line: fmt.Sprintf("%s: %s", pos, diag.Message)}); err != nil {
				// Losing the finding would be worse than mixing it into stdout.
				pass.Report(diag)

				return
			}

			// Single summary line on stderr so the caller knows to check the file.
			writer.summaryOnce.Do(func() {
				_, _ = fmt.Fprintf(os.Stderr, "[agent] heuristic findings written to %s\n", writer.path)
			})
		}

		return originalRun(&filteredPass)
	}

	return &wrapped
}
