package analyzers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/tools/go/analysis"
)

// agentFileMode keeps findings readable only by the user who ran the linter,
// since the file sits in a shared temp directory.
const agentFileMode = 0o600

// agentFileWriter appends agent-tier diagnostics to a per-process temp file,
// keeping them out of the stdout stream humans and CI read. It holds no open
// handle: the framework has no end-of-run hook to close one, and opening at
// construction would leave an empty file after a clean run. Each line opens,
// writes, and closes the file under mu instead.
type agentFileWriter struct {
	path        string
	mu          sync.Mutex
	summaryOnce sync.Once
	opened      bool
}

func newAgentFileWriter() *agentFileWriter {
	return &agentFileWriter{
		path: filepath.Join(os.TempDir(), fmt.Sprintf("cairnlint-agent-%d.txt", os.Getpid())),
	}
}

// writeLine appends line to the agent file, creating it on the first call.
// The first open also truncates, so a file an earlier process with the same
// PID left behind does not carry its findings into this run.
func (writer *agentFileWriter) writeLine(line string) error {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	flags := os.O_APPEND | os.O_CREATE | os.O_WRONLY
	if !writer.opened {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(writer.path, flags, agentFileMode)
	if err != nil {
		return fmt.Errorf("open agent file: %w", err)
	}

	writer.opened = true

	_, writeErr := fmt.Fprintln(file, line)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return fmt.Errorf("write agent file %s: %w", writer.path, err)
	}

	return nil
}

// wrapAgentFileOutput redirects agent diagnostics to a temp file and prints
// one stderr line on the first finding so the caller knows where to look. A
// diagnostic the file cannot take falls back to stdout. It replaces
// pass.Report without forwarding, so it must be the outermost wrapper; enter
// through WrapForAgentFile.
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
			if err := writer.writeLine(fmt.Sprintf("%s: %s", pos, diag.Message)); err != nil {
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
