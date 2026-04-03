package analyzers

import (
	"go/token"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// taskMarkerPattern matches comment lines containing task markers
// (the marker keyword followed by a word boundary).
var taskMarkerPattern = regexp.MustCompile(`\b(TODO|FIXME|HACK|XXX)\b`)

// attributionPattern matches task markers that have an owner (name) or
// a ticket reference like PROJ-123 immediately after the keyword.
var attributionPattern = regexp.MustCompile(`\b(TODO|FIXME|HACK|XXX)\s*(\([^)]+\)|[A-Z]+-\d+)`)

// unattributedTODOAnalyzer returns an analyzer that flags comment lines
// containing task markers without an owner or ticket reference. Every marker
// should have either a parenthesized owner name like TODO(alice) or a ticket
// reference like TODO PROJ-123 so responsibility is traceable.
func unattributedTODOAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "unattributedtodo",
		Doc:  "flags TODO/FIXME/HACK/XXX without owner (name) or ticket reference PROJ-123",
		Run:  runUnattributedTODO,
	}
}

func runUnattributedTODO(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		for _, cg := range file.Comments {
			for _, comment := range cg.List {
				checkCommentForUnattributedMarker(pass, comment.Pos(), comment.Text)
			}
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkCommentForUnattributedMarker inspects a single comment line for
// task markers without attribution and reports a diagnostic if found.
func checkCommentForUnattributedMarker(pass *analysis.Pass, pos token.Pos, text string) {
	content := strings.TrimPrefix(text, "//")
	if strings.HasPrefix(text, "/*") {
		content = strings.TrimSuffix(text[2:], "*/")
	}

	if !taskMarkerPattern.MatchString(content) {
		return
	}

	if attributionPattern.MatchString(content) {
		return
	}

	pass.Reportf(pos, "task marker without owner or ticket; use TODO(name) or TODO PROJ-123")
}
