package analyzers

import (
	"golang.org/x/tools/go/analysis"
)

// noDotImportAnalyzer returns an analyzer that flags dot imports
// (import . "pkg"). Dot imports drop the package qualifier, which hides where
// names come from and invites collisions. Qualify package references instead.
func noDotImportAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "nodotimport",
		Doc:  "flags dot imports (import . \"pkg\"); qualify package references explicitly",
		Run:  runNoDotImport,
	}
}

func runNoDotImport(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		for _, imp := range file.Imports {
			if imp.Name != nil && imp.Name.Name == "." {
				pass.Reportf(imp.Pos(), "dot import %s; qualify the package explicitly instead", imp.Path.Value)
			}
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
