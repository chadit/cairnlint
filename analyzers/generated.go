package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis"
)

// maxGeneratedHeaderBytes bounds how much of a //line target is read to find
// its package clause, so a target that is huge or never ends cannot stall
// the run.
const maxGeneratedHeaderBytes = 64 << 10 // 64 KiB

// generatedCache memoizes the generated verdict per *ast.File, because every
// wrapped analyzer rebuilds the generated set for each package it runs on
// and a //line-redirected file costs a disk read to classify.
//
//nolint:gochecknoglobals // process-wide cache shared by every wrapped analyzer
var generatedCache sync.Map

// WrapSkipGenerated wraps each analyzer so diagnostics in generated files
// are suppressed. Generated files are identified by the standard
// "Code generated ... DO NOT EDIT." header that ast.IsGenerated recognizes,
// a .pb.go suffix, or a /gen/ path.
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

// buildGeneratedSet returns the position-adjusted filenames of generated
// files in the pass, matching the filenames diagnostics report under.
func buildGeneratedSet(pass *analysis.Pass) map[string]bool {
	result := make(map[string]bool, len(pass.Files))

	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename
		if cachedIsGenerated(pass.Fset, filename, file) {
			result[filename] = true
		}
	}

	return result
}

// cachedIsGenerated serves isGenerated from generatedCache.
func cachedIsGenerated(fset *token.FileSet, filename string, file *ast.File) bool {
	if cached, ok := generatedCache.Load(file); ok {
		if generated, ok := cached.(bool); ok {
			return generated
		}
	}

	generated := isGenerated(fset, filename, file)
	generatedCache.Store(file, generated)

	return generated
}

// isGenerated checks path conventions and then the standard header. When a
// //line directive redirects the file's positions to another Go file, as
// cmd/cgo output does, the header in the parsed file belongs to the
// preprocessor and not to the source the user wrote, so the header check
// moves to the //line target. Without that, every hand-written cgo file in
// a package went silent. A redirect to a non-Go source, as goyacc output
// does with its grammar, keeps the parsed file's own header: the Go file
// is the generated artifact there.
func isGenerated(fset *token.FileSet, filename string, file *ast.File) bool {
	if strings.HasSuffix(filename, ".pb.go") || strings.Contains(filename, "/gen/") {
		return true
	}

	if fset.File(file.FileStart).Name() == filename || !strings.HasSuffix(filename, ".go") {
		return ast.IsGenerated(file)
	}

	return sourceHasGeneratedHeader(filename, file)
}

// sourceHasGeneratedHeader reports whether the Go file at filename starts
// with a generated-code header. When the file cannot be read or parsed the
// verdict falls back to the header of redirected, the parsed file that
// pointed here, so a dangling //line target never un-generates a file that
// says it is generated.
func sourceHasGeneratedHeader(filename string, redirected *ast.File) bool {
	header, ok := readHeader(filename)
	if !ok {
		return ast.IsGenerated(redirected)
	}

	return ast.IsGenerated(header)
}

// readHeader parses the package clause and the comments before it from the
// regular file at filename, reading at most maxGeneratedHeaderBytes. The
// file sits outside the pass's own file list, which is why pass.ReadFile
// cannot serve it. It is opened through an os.Root pinned to its directory
// so the //line name cannot walk anywhere else, and the regular-file check
// keeps a //line pointing at a device or FIFO from blocking the run.
func readHeader(filename string) (*ast.File, bool) {
	root, err := os.OpenRoot(filepath.Dir(filename))
	if err != nil {
		return nil, false
	}

	defer func() { _ = root.Close() }()

	source, err := root.Open(filepath.Base(filename))
	if err != nil {
		return nil, false
	}

	defer func() { _ = source.Close() }()

	info, err := source.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, false
	}

	src, err := io.ReadAll(io.LimitReader(source, maxGeneratedHeaderBytes))
	if err != nil {
		return nil, false
	}

	header, err := parser.ParseFile(token.NewFileSet(), filename, src, parser.PackageClauseOnly|parser.ParseComments)
	if err != nil {
		return nil, false
	}

	return header, true
}
