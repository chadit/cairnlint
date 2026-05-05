package analyzers

import (
	"go/ast"
	"strings"
	"unicode"

	"golang.org/x/tools/go/analysis"
)

// agentExportedInTestFileAnalyzer flags exported declarations in augmented
// _test.go files (same-package, using package foo instead of package foo_test).
// These add symbols to the package API surface solely during compilation,
// which usually means they exist only for verification access.
//
// This is an agent-only analyzer because legitimate patterns exist (e.g.
// framework entry points, Example functions) and some exported helpers are
// intentional. An LLM can grep for usage and decide; a human would get
// tired of dismissing false positives.
func agentExportedInTestFileAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "agentexportedintestfile",
		Doc:  "[agent] flags exported declarations in augmented test files (same-package _test.go)",
		Run:  runAgentExportedInTestFile,
	}
}

func runAgentExportedInTestFile(pass *analysis.Pass) (any, error) {
	pkgName := pass.Pkg.Name()

	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename

		if !strings.HasSuffix(filename, "_test.go") {
			continue
		}

		// Only flag augmented test files (package foo, not package foo_test).
		// External test packages are fine, their exports are test-internal.
		if strings.HasSuffix(pkgName, "_test") {
			continue
		}

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				checkExportedFunc(pass, d)
			case *ast.GenDecl:
				checkExportedGenDecl(pass, d)
			}
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkExportedFunc flags exported functions that aren't test/benchmark/
// fuzz/example functions or TestMain.
func checkExportedFunc(pass *analysis.Pass, funcDecl *ast.FuncDecl) {
	name := funcDecl.Name.Name
	if !isExportedName(name) {
		return
	}

	// Framework entry points (TestXxx, BenchmarkXxx, etc.) are expected here.
	if isTestFrameworkFunc(name) {
		return
	}

	kind := "func"
	if funcDecl.Recv != nil {
		kind = "method"
	}

	pass.Reportf(funcDecl.Name.Pos(),
		"[agent] exported %s %s in augmented _test.go file: adds to package API only during compilation; verify this isn't solely for verification access",
		kind, name)
}

// checkExportedGenDecl flags exported var, const, and type declarations.
func checkExportedGenDecl(pass *analysis.Pass, decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		switch typedSpec := spec.(type) {
		case *ast.ValueSpec:
			for _, ident := range typedSpec.Names {
				if isExportedName(ident.Name) {
					pass.Reportf(ident.Pos(),
						"[agent] exported var/const %s in augmented _test.go file: adds to package API only during compilation; verify this isn't solely for verification access",
						ident.Name)
				}
			}
		case *ast.TypeSpec:
			if isExportedName(typedSpec.Name.Name) {
				pass.Reportf(typedSpec.Name.Pos(),
					"[agent] exported type %s in augmented _test.go file: adds to package API only during compilation; verify this isn't solely for verification access",
					typedSpec.Name.Name)
			}
		}
	}
}

func isExportedName(name string) bool {
	if name == "" {
		return false
	}

	return unicode.IsUpper(rune(name[0]))
}

// isTestFrameworkFunc returns true for function names that the Go test
// framework expects: TestXxx, BenchmarkXxx, FuzzXxx, ExampleXxx, and
// TestMain.
func isTestFrameworkFunc(name string) bool {
	if name == "TestMain" {
		return true
	}

	prefixes := []string{"Test", "Benchmark", "Fuzz", "Example"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) && len(name) > len(prefix) {
			// Go requires the character after the prefix to be uppercase
			// or non-letter (e.g. Test_foo is valid but unusual).
			return true
		}
	}

	return false
}
