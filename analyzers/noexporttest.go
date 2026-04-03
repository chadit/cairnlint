package analyzers

import (
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// noExportTestAnalyzer returns an analyzer that flags files named
// export_test.go. These files expose package internals to external test
// packages. If tests need unexported symbols, the public API is incomplete
// and should be fixed instead.
func noExportTestAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "noexporttest",
		Doc:  "flags export_test.go files; redesign the public API instead of exposing internals",
		Run:  runNoExportTest,
	}
}

func runNoExportTest(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename
		base := filepath.Base(filename)

		if strings.EqualFold(base, "export_test.go") {
			pass.Reportf(file.Name.Pos(), "export_test.go not allowed; redesign the public API so tests don't need internal access")
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
