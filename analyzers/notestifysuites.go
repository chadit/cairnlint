package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// noTestifySuitesAnalyzer returns an analyzer that flags struct types
// embedding suite.Suite from the testify library. Test suites add
// indirection and hidden state; standalone test functions with
// require/assert are simpler and easier to debug.
func noTestifySuitesAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "notestifysuites",
		Doc:      "flags suite.Suite embedding in test structs; use standalone test functions with require/assert",
		Run:      runNoTestifySuites,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runNoTestifySuites(pass *analysis.Pass) (any, error) {
	if !hasTestFiles(pass) {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		typeSpec, isTypeSpec := node.(*ast.TypeSpec)
		if !isTypeSpec {
			return
		}

		if !isTestFile(pass, typeSpec) {
			return
		}

		structType, isStruct := typeSpec.Type.(*ast.StructType)
		if !isStruct || structType.Fields == nil {
			return
		}

		for _, field := range structType.Fields.List {
			if isSuiteSuiteEmbed(field) {
				pass.Reportf(field.Pos(), "do not embed suite.Suite; use standalone test functions with require/assert")
			}
		}
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isSuiteSuiteEmbed reports whether field is an embedded suite.Suite field.
// Embedded fields have no names, and the type is a selector expression
// suite.Suite.
func isSuiteSuiteEmbed(field *ast.Field) bool {
	// Embedded fields have zero explicit names.
	if len(field.Names) != 0 {
		return false
	}

	sel, isSel := field.Type.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != "Suite" {
		return false
	}

	ident, isIdent := sel.X.(*ast.Ident)

	return isIdent && ident.Name == "suite"
}
