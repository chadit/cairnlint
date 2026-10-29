package analyzers

import (
	"go/ast"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// mainFuncName is the function the runtime calls in package main.
const mainFuncName = "main"

// noPanicInLibAnalyzer returns an analyzer that flags panic() calls in
// non-test files, because library code should return errors. A panic directly
// in init, or in main of package main, is exempt: startup code has no caller
// to return an error to. A closure inside either is still flagged, since it
// goes to a handler or goroutine and runs as library code.
func noPanicInLibAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "nopanicinlib",
		Doc:      "flags panic() in non-test files outside main() and init(); library code should return errors",
		Run:      runNoPanicInLib,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runNoPanicInLib(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.WithStack(nodeFilter, func(node ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		call, isCall := node.(*ast.CallExpr)

		// Skip test files where panic is acceptable for test helpers.
		if !isCall || isTestFile(pass, call) {
			return true
		}

		ident, isIdent := call.Fun.(*ast.Ident)
		if !isIdent || ident.Name != "panic" || inProgramEntry(pass.Pkg, stack) {
			return true
		}

		pass.Reportf(call.Pos(), "panic() in library code; return an error instead")

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// inProgramEntry reports whether the innermost function around the node on
// top of stack is init, or main in package main. A function literal ends the
// search, so a closure inside either one does not inherit the exemption.
func inProgramEntry(pkg *types.Package, stack []ast.Node) bool {
	for _, node := range slices.Backward(stack) {
		switch fn := node.(type) {
		case *ast.FuncLit:
			return false
		case *ast.FuncDecl:
			return isProgramEntryDecl(pkg, fn)
		}
	}

	return false
}

// isProgramEntryDecl reports whether decl is a package-level init, or main in
// package main. A method named either is ordinary code.
func isProgramEntryDecl(pkg *types.Package, decl *ast.FuncDecl) bool {
	if decl.Recv != nil {
		return false
	}

	switch decl.Name.Name {
	case "init":
		return true
	case mainFuncName:
		return pkg.Name() == mainPkgName
	}

	return false
}
