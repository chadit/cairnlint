package analyzers

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"golang.org/x/tools/go/analysis"
)

// tabwriterPadding is the minimum padding inserted between the analyzer
// name column and the doc-string column in `cairnlint linters`.
const tabwriterPadding = 2

// PrintLinters writes a human-readable list of every analyzer to out,
// grouped by category. Agent-only analyzers are appended under their own
// heading and tagged [agent]. Output is similar in spirit to
// `golangci-lint linters`.
func PrintLinters(out io.Writer) error {
	writer := tabwriter.NewWriter(out, 0, 0, tabwriterPadding, ' ', 0)
	errw := &errWriter{w: writer}

	errw.printf("Available linters:\n\n")

	for _, cat := range Categories() {
		errw.printf("%s:\n", cat.Name)
		writeAnalyzerRows(errw, cat.Analyzers, "")
		errw.printf("\n")
	}

	if agents := AgentOnly(); len(agents) > 0 {
		errw.printf("Agent-only (enable with --agent or auto-detected):\n")
		writeAnalyzerRows(errw, agents, "")
		errw.printf("\n")
	}

	errw.printf("Suppress diagnostics with //nolint:<name> on the offending line,\n")
	errw.printf("or //nolint:<name> on the line above a statement or function.\n")

	if errw.err != nil {
		return fmt.Errorf("print linters: %w", errw.err)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush linters output: %w", err)
	}

	return nil
}

func writeAnalyzerRows(errw *errWriter, list []*analysis.Analyzer, prefix string) {
	for _, a := range list {
		errw.printf("  %s\t%s%s\n", a.Name, prefix, firstLine(a.Doc))
	}
}

// errWriter wraps an io.Writer and records the first error seen so callers
// can issue many Fprintf calls without checking each one individually.
type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) printf(format string, a ...any) {
	if e.err != nil {
		return
	}

	_, e.err = fmt.Fprintf(e.w, format, a...)
}

// firstLine returns the first line of doc, which for analysis.Analyzer.Doc
// is typically the short summary shown in the linters listing.
func firstLine(doc string) string {
	head, _, _ := strings.Cut(doc, "\n")

	return head
}
