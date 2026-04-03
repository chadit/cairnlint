package analyzers

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// sentinelErrorsAnalyzer returns an analyzer that flags sentinel error
// declarations (var ErrFoo = errors.New(...)) in files not named errors.go.
// Centralizing sentinel errors in errors.go makes them discoverable and
// keeps each package's error surface in one place.
func sentinelErrorsAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "sentinelerrors",
		Doc:  "flags sentinel error vars outside errors.go; centralize them for discoverability",
		Run:  runSentinelErrors,
	}
}

func runSentinelErrors(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename

		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		if strings.HasSuffix(filename, "errors.go") {
			continue
		}

		checkFileSentinelErrors(pass, file)
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkFileSentinelErrors walks var declarations in a single file and
// reports any that look like sentinel error definitions.
func checkFileSentinelErrors(pass *analysis.Pass, file *ast.File) {
	for _, decl := range file.Decls {
		genDecl, isGen := decl.(*ast.GenDecl)
		if !isGen || genDecl.Tok != token.VAR {
			continue
		}

		reportSentinelSpecs(pass, genDecl)
	}
}

// reportSentinelSpecs checks each spec in a var block for sentinel error
// patterns and reports violations.
func reportSentinelSpecs(pass *analysis.Pass, genDecl *ast.GenDecl) {
	for _, spec := range genDecl.Specs {
		valSpec, isVal := spec.(*ast.ValueSpec)
		if !isVal {
			continue
		}

		reportSentinelNames(pass, valSpec)
	}
}

// reportSentinelNames flags individual var names that match sentinel
// error naming and are initialized with errors.New().
func reportSentinelNames(pass *analysis.Pass, valSpec *ast.ValueSpec) {
	for idx, name := range valSpec.Names {
		if !isSentinelErrorName(name.Name) {
			continue
		}

		if idx < len(valSpec.Values) && isErrorsNewCall(valSpec.Values[idx]) {
			pass.Reportf(name.Pos(), "sentinel error %s should be declared in errors.go", name.Name)
		}
	}
}

// isSentinelErrorName reports whether name follows the sentinel error naming
// convention: starts with "Err" (exported) or "err" followed by an uppercase
// letter (unexported).
func isSentinelErrorName(name string) bool {
	if strings.HasPrefix(name, "Err") && len(name) > 3 {
		return true
	}

	if strings.HasPrefix(name, "err") && len(name) > 3 {
		ch := name[3]
		if ch >= 'A' && ch <= 'Z' {
			return true
		}
	}

	return false
}

// isErrorsNewCall reports whether expr is a call to errors.New().
func isErrorsNewCall(expr ast.Expr) bool {
	call, isCall := expr.(*ast.CallExpr)
	if !isCall {
		return false
	}

	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel {
		return false
	}

	ident, isIdent := sel.X.(*ast.Ident)
	if !isIdent {
		return false
	}

	return ident.Name == "errors" && sel.Sel.Name == "New"
}
