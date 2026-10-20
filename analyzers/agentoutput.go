package analyzers

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/tools/go/analysis"
)

// agentFileWriter buffers agent-tier diagnostics to a temp file so they stay
// out of the stdout stream humans and CI read.
//
// The pointer-bearing fields lead so the garbage collector stops scanning
// after the string header, which is the ordering govet's fieldalignment check
// wants. The mutex still guards every field below it.
type agentFileWriter struct {
	file        *os.File
	path        string
	mu          sync.Mutex
	summaryOnce sync.Once
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

// wrapAgentFileOutput redirects agent diagnostics to a temp file and prints
// one stderr line on the first finding so the caller knows where to look. If
// the file cannot be created, diagnostics fall back to stdout. It replaces
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
