package analyzers

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/tools/go/analysis"
)

type agentFileWriter struct {
	mu          sync.Mutex
	summaryOnce sync.Once
	path        string
	file        *os.File
}

func newAgentFileWriter() *agentFileWriter {
	writer := &agentFileWriter{}
	writer.path = filepath.Join(os.TempDir(), fmt.Sprintf("cairnlint-agent-%d.txt", os.Getpid()))

	fileHandle, err := os.Create(writer.path)
	if err != nil {
		return writer
	}

	writer.file = fileHandle

	return writer
}

func (writer *agentFileWriter) writeLine(line string) {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if writer.file == nil {
		return
	}

	_, _ = fmt.Fprintln(writer.file, line)
}

func (writer *agentFileWriter) filePath() string {
	return writer.path
}

// WrapAgentFileOutput wraps agent analyzers to redirect their diagnostics
// to a temp file. Diagnostics are suppressed from stdout. A single summary
// line is printed to stderr on the first finding so the caller (human or
// LLM) knows where to look.
//
// When the file cannot be created, diagnostics pass through to stdout as
// a fallback.
func WrapAgentFileOutput(all []*analysis.Analyzer) []*analysis.Analyzer {
	writer := newAgentFileWriter()

	wrapped := make([]*analysis.Analyzer, len(all))
	for idx, orig := range all {
		wrapped[idx] = wrapAgentOut(orig, writer)
	}

	return wrapped
}

func wrapAgentOut(orig *analysis.Analyzer, writer *agentFileWriter) *analysis.Analyzer {
	originalRun := orig.Run

	wrapped := *orig
	wrapped.Run = func(pass *analysis.Pass) (any, error) {
		// If file creation failed, fall through to normal output.
		if writer.file == nil {
			return originalRun(pass)
		}

		filteredPass := *pass

		filteredPass.Report = func(diag analysis.Diagnostic) {
			pos := pass.Fset.Position(diag.Pos)
			writer.writeLine(fmt.Sprintf("%s: %s", pos, diag.Message))

			// Single summary line on stderr so the caller knows to check the file.
			writer.summaryOnce.Do(func() {
				_, _ = fmt.Fprintf(os.Stderr, "[agent] heuristic findings written to %s\n", writer.filePath())
			})
		}

		return originalRun(&filteredPass)
	}

	return &wrapped
}
