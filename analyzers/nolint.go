package analyzers

import (
	"go/ast"
	"go/token"
	"regexp"
	"slices"
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis"
)

// suppressPattern matches a nolint directive at the start of a line comment,
// following golangci-lint: after `//` and optional spaces the word `nolint`
// must be followed by a colon, whitespace, or the end of the comment, so
// `//nolintfoo`, `//nolint-foo`, and `nolint` mentioned mid-sentence are not
// directives. The name list after the colon runs to the next `//`, which is
// how golangci-lint separates the list from its explanation, so
// `//nolint:a/b, c // why` yields `a/b, c`. The capture keeps the colon so an
// explicit empty list (`//nolint:`) can be told apart from a bare `//nolint`.
var suppressPattern = regexp.MustCompile(`^//\s*nolint(?:(:(?:[^/]|/[^/])*)|\s|$)`)

// suppressAllName is golangci-lint's wildcard name that suppresses every analyzer.
const suppressAllName = "all"

// fileLine identifies one source line across the files of a pass. Keying
// suppression by line number alone let a directive in one file silence the
// same line number in every other file of the package.
type fileLine struct {
	filename string
	line     int
}

// WrapWithNolint wraps each analyzer so diagnostics suppressed by a matching
// //nolint directive are dropped. This gives standalone cairnlint the same
// suppression behavior as golangci-lint: a trailing //nolint on a line
// suppresses that line, a //nolint directly above a statement suppresses
// the whole statement, and one above a function suppresses the entire
// function body.
func WrapWithNolint(all []*analysis.Analyzer) []*analysis.Analyzer {
	wrapped := make([]*analysis.Analyzer, len(all))
	for idx, orig := range all {
		wrapped[idx] = wrapAnalyzer(orig)
	}

	return wrapped
}

func wrapAnalyzer(orig *analysis.Analyzer) *analysis.Analyzer {
	originalRun := orig.Run

	wrapped := *orig
	wrapped.Run = func(pass *analysis.Pass) (any, error) {
		nolintLines := buildNolintMap(pass, orig.Name)

		originalPass := *pass
		originalReport := pass.Report

		originalPass.Report = func(diag analysis.Diagnostic) {
			pos := pass.Fset.Position(diag.Pos)
			if nolintLines[fileLine{filename: pos.Filename, line: pos.Line}] {
				return
			}

			originalReport(diag)
		}

		return originalRun(&originalPass)
	}

	return &wrapped
}

// directive is one parsed //nolint comment plus the line range it suppresses.
// names holds the lowercased analyzer names from `//nolint:a,b,c`; all is set
// for a bare `//nolint` or `//nolint:all`. filename is the position-adjusted
// name, so a //line directive moves the suppression along with the
// diagnostics it covers. filename leads names because govet's fieldalignment
// wants the pointer-bearing bytes packed at the front.
type directive struct {
	filename  string
	names     []string
	startLine int
	endLine   int
	all       bool
}

// applies reports whether the directive suppresses analyzerName.
func (d directive) applies(analyzerName string) bool {
	return d.all || slices.Contains(d.names, analyzerName)
}

// fileDirectiveCache memoizes parsed directives per *ast.File so the full
// AST walk in ast.NewCommentMap runs once per file instead of once per
// wrapped analyzer. Keys are stable for the lifetime of a multichecker
// invocation; the cache is intentionally never evicted.
//
//nolint:gochecknoglobals // process-wide cache shared by every wrapped analyzer
var fileDirectiveCache sync.Map

// directivesFor returns every //nolint directive in file, computing on the
// first call and serving subsequent callers from the cache.
func directivesFor(fset *token.FileSet, file *ast.File) []directive {
	if cached, ok := fileDirectiveCache.Load(file); ok {
		if list, ok := cached.([]directive); ok {
			return list
		}
	}

	list := collectDirectives(fset, file)
	actual, _ := fileDirectiveCache.LoadOrStore(file, list)

	if stored, ok := actual.([]directive); ok {
		return stored
	}

	return list
}

// span is the line range one directive comment suppresses.
type span struct {
	start   token.Position
	endLine int
}

// collectDirectives walks file once, gathering both the trailing-comment
// form (line range is a single line) and the leading-comment form (line
// range spans the associated node so whole-function suppression works).
func collectDirectives(fset *token.FileSet, file *ast.File) []directive {
	var list []directive

	for _, cg := range file.Comments {
		list = appendDirectives(list, cg, func(comment *ast.Comment) span {
			pos := fset.Position(comment.Pos())

			return span{start: pos, endLine: pos.Line}
		})
	}

	// ast.NewCommentMap may attach one comment group to multiple nodes
	// (e.g., a FuncDecl and its inner BlockStmt). We record a directive
	// entry for every attachment and let the union in buildNolintMap take
	// the widest span. This is a superset of strict golangci-lint
	// semantics: a leading directive occasionally suppresses a line or
	// two more than a reader might expect, but never less.
	cmap := ast.NewCommentMap(fset, file, file.Comments)
	for node, groups := range cmap {
		nodeSpan := span{start: fset.Position(node.Pos()), endLine: fset.Position(node.End()).Line}

		for _, cg := range groups {
			list = appendDirectives(list, cg, func(*ast.Comment) span { return nodeSpan })
		}
	}

	return list
}

// appendDirectives adds one directive per //nolint comment in cg, covering
// the span that spanOf assigns to that comment.
func appendDirectives(list []directive, cg *ast.CommentGroup, spanOf func(*ast.Comment) span) []directive {
	for _, comment := range cg.List {
		names, all, ok := parseDirective(comment.Text)
		if !ok {
			continue
		}

		lines := spanOf(comment)
		list = append(list, directive{
			names:     names,
			filename:  lines.start.Filename,
			startLine: lines.start.Line,
			endLine:   lines.endLine,
			all:       all,
		})
	}

	return list
}

// buildNolintMap returns the set of file lines suppressed by a matching
// //nolint directive for analyzerName. The heavy work of parsing directives
// and building the comment map is cached per file, so this function is
// cheap to call from each wrapped analyzer.
func buildNolintMap(pass *analysis.Pass, analyzerName string) map[fileLine]bool {
	result := make(map[fileLine]bool)

	for _, file := range pass.Files {
		for _, suppression := range directivesFor(pass.Fset, file) {
			markSuppressed(result, suppression, analyzerName)
		}
	}

	return result
}

// markSuppressed records every line of suppression into result when the
// directive applies to analyzerName.
func markSuppressed(result map[fileLine]bool, suppression directive, analyzerName string) {
	if !suppression.applies(analyzerName) {
		return
	}

	for line := suppression.startLine; line <= suppression.endLine; line++ {
		result[fileLine{filename: suppression.filename, line: line}] = true
	}
}

// parseDirective extracts the lowercased analyzer names from a //nolint
// comment and reports whether the directive covers every analyzer. The
// third result is false when text is not a directive. An explicit empty
// list (`//nolint:`) is a directive that names nothing, which golangci-lint
// treats as matching no linter.
func parseDirective(text string) ([]string, bool, bool) {
	if !strings.Contains(text, "nolint") {
		return nil, false, false
	}

	match := suppressPattern.FindStringSubmatch(text)
	if match == nil {
		return nil, false, false
	}

	if match[1] == "" {
		return nil, true, true
	}

	var names []string

	for name := range strings.SplitSeq(strings.TrimPrefix(match[1], ":"), ",") {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == suppressAllName {
			return nil, true, true
		}

		if name != "" {
			names = append(names, name)
		}
	}

	return names, false, true
}
