package analyzers

import (
	"strings"

	"golang.org/x/tools/go/analysis"
)

// externalTestPkgAnalyzer returns an analyzer that flags test files using
// internal test packages. Test files should use package foo_test (external)
// so they exercise the public API and expose missing exported surface.
func externalTestPkgAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "externaltestpkg",
		Doc:  "flags test files that use internal test packages; use package name_test instead",
		Run:  runExternalTestPkg,
	}
}

func runExternalTestPkg(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename

		if !strings.HasSuffix(filename, "_test.go") {
			continue
		}

		pkgName := file.Name.Name
		if !strings.HasSuffix(pkgName, "_test") {
			pass.Reportf(file.Name.Pos(), "test file must use external test package (package %s_test, not %s)", pkgName, pkgName)
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
