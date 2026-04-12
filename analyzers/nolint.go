package analyzers

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis"
)

// suppressPattern matches nolint directives in comments with optional
// analyzer names (bare, single, or comma-separated). The `\b` after
// `nolint` prevents false matches against identifiers like `nolintfoo`;
// the name character class rejects accidental trailing junk such as
// `//nolint:foo//bar` (which would previously parse as the name list
// `foo//bar`).
var suppressPattern = regexp.MustCompile(`//\s*nolint\b(?::([A-Za-z0-9_,\-]+))?`)

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
			line := pass.Fset.Position(diag.Pos).Line
			if nolintLines[line] {
				return
			}

			originalReport(diag)
		}

		return originalRun(&originalPass)
	}

	return &wrapped
}

// directive is one parsed //nolint comment plus the line range it suppresses.
// names is the raw comma-separated list from `//nolint:a,b,c`; an empty
// string means a bare `//nolint` (applies to every analyzer).
type directive struct {
	names     string
	startLine int
	endLine   int
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

// collectDirectives walks file once, gathering both the trailing-comment
// form (line range is a single line) and the leading-comment form (line
// range spans the associated node so whole-function suppression works).
func collectDirectives(fset *token.FileSet, file *ast.File) []directive {
	var list []directive

	for _, cg := range file.Comments {
		for _, comment := range cg.List {
			names, ok := parseDirective(comment.Text)
			if !ok {
				continue
			}

			line := fset.Position(comment.Pos()).Line
			list = append(list, directive{names: names, startLine: line, endLine: line})
		}
	}

	// ast.NewCommentMap may attach one comment group to multiple nodes
	// (e.g., a FuncDecl and its inner BlockStmt). We record a directive
	// entry for every attachment and let the union in buildNolintMap take
	// the widest span. This is a superset of strict golangci-lint
	// semantics: a leading directive occasionally suppresses a line or
	// two more than a reader might expect, but never less.
	cmap := ast.NewCommentMap(fset, file, file.Comments)
	for node, groups := range cmap {
		start := fset.Position(node.Pos()).Line
		end := fset.Position(node.End()).Line

		for _, cg := range groups {
			for _, comment := range cg.List {
				names, ok := parseDirective(comment.Text)
				if !ok {
					continue
				}

				list = append(list, directive{names: names, startLine: start, endLine: end})
			}
		}
	}

	return list
}

// buildNolintMap returns the set of line numbers suppressed by a matching
// //nolint directive for analyzerName. The heavy work of parsing directives
// and building the comment map is cached per file, so this function is
// cheap to call from each wrapped analyzer.
func buildNolintMap(pass *analysis.Pass, analyzerName string) map[int]bool {
	result := make(map[int]bool) //nolint:mapprealloc // size depends on nolint comment density, not file count

	for _, file := range pass.Files {
		for _, d := range directivesFor(pass.Fset, file) {
			if !namesApply(d.names, analyzerName) {
				continue
			}

			for line := d.startLine; line <= d.endLine; line++ {
				result[line] = true
			}
		}
	}

	return result
}

// parseDirective extracts the raw name list from a //nolint comment, or
// reports !ok when text is not a directive. An empty names string means
// a bare directive (suppresses every analyzer).
func parseDirective(text string) (string, bool) {
	if !strings.Contains(text, "nolint") {
		return "", false
	}

	match := suppressPattern.FindStringSubmatch(text)
	if match == nil {
		return "", false
	}

	return match[1], true
}

// namesApply reports whether a directive's name list applies to analyzerName.
// An empty list means bare //nolint and applies to every analyzer.
func namesApply(names, analyzerName string) bool {
	if names == "" {
		return true
	}

	for name := range strings.SplitSeq(names, ",") {
		if strings.TrimSpace(name) == analyzerName {
			return true
		}
	}

	return false
}
