package analyzers

import (
	"strings"

	"golang.org/x/tools/go/analysis"
)

// testCryptoInProdAnalyzer returns an analyzer that flags imports of test-only
// crypto packages in non-test files. Packages like crypto/mlkem/mlkemtest and
// testing/cryptotest provide deterministic crypto for testing and must never
// appear in production code.
func testCryptoInProdAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "testcryptoinprod",
		Doc:  "flags test-only crypto package imports in non-test files",
		Run:  runTestCryptoInProd,
	}
}

// isTestOnlyCryptoPackage reports whether importPath is a package that provides
// deterministic crypto primitives intended exclusively for test code.
func isTestOnlyCryptoPackage(importPath string) bool {
	switch importPath {
	case "crypto/mlkem/mlkemtest", "testing/cryptotest":
		return true
	default:
		return false
	}
}

func runTestCryptoInProd(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename

		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		for _, importSpec := range file.Imports {
			importPath := strings.Trim(importSpec.Path.Value, "\"")

			if isTestOnlyCryptoPackage(importPath) {
				pass.Reportf(importSpec.Pos(), "test-only crypto package %s must not be imported in production code", importPath)
			}
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
