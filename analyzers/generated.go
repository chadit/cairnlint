package analyzers

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// WrapSkipGenerated wraps each analyzer so diagnostics in generated files
// are suppressed. Generated files are identified by the standard
// "Code generated ... DO NOT EDIT." marker, .pb.go suffix, or /gen/ path.
func WrapSkipGenerated(all []*analysis.Analyzer) []*analysis.Analyzer {
	wrapped := make([]*analysis.Analyzer, len(all))
	for idx, orig := range all {
		wrapped[idx] = wrapSkipGen(orig)
	}

	return wrapped
}

func wrapSkipGen(orig *analysis.Analyzer) *analysis.Analyzer {
	originalRun := orig.Run

	wrapped := *orig
	wrapped.Run = func(pass *analysis.Pass) (any, error) {
		genFiles := buildGeneratedSet(pass)
		if len(genFiles) == 0 {
			return originalRun(pass)
		}

		filteredPass := *pass
		originalReport := pass.Report

		filteredPass.Report = func(diag analysis.Diagnostic) {
			if genFiles[pass.Fset.Position(diag.Pos).Filename] {
				return
			}

			originalReport(diag)
		}

		return originalRun(&filteredPass)
	}

	return &wrapped
}

// buildGeneratedSet returns filenames of generated files in the pass.
func buildGeneratedSet(pass *analysis.Pass) map[string]bool {
	result := make(map[string]bool)

	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename
		if isGenerated(filename, file) {
			result[filename] = true
		}
	}

	return result
}

// isGenerated checks path patterns and the standard generated-code marker.
func isGenerated(filename string, file *ast.File) bool {
	if strings.HasSuffix(filename, ".pb.go") || strings.Contains(filename, "/gen/") {
		return true
	}

	for _, cg := range file.Comments {
		for _, comment := range cg.List {
			if strings.Contains(comment.Text, "Code generated") && strings.Contains(comment.Text, "DO NOT EDIT") {
				return true
			}
		}
	}

	return false
}
