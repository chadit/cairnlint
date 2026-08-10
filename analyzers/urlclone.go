package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// URL types whose copies need the Go 1.27 Clone methods to be complete.
const (
	urlValuesType = "Values"
	urlURLType    = "URL"
)

// urlCloneAnalyzer returns an analyzer that flags copies of net/url values
// that share state with the original. Go 1.27 added URL.Clone and
// Values.Clone, which copy all the way down.
//
// Both shapes look finished and are not. maps.Clone on url.Values duplicates
// the map while every []string value stays shared, so appending to a copied
// key mutates the original. Dereferencing a *url.URL copies the struct while
// the *Userinfo pointer stays shared. Neither aliasing bug shows up until
// something writes through the copy.
func urlCloneAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "urlclone",
		Doc:      "flags shallow copies of url.Values and url.URL; use the Clone methods instead (Go 1.27+)",
		Run:      runURLClone,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runURLClone(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
		(*ast.StarExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		switch expr := node.(type) {
		case *ast.CallExpr:
			reportMapsCloneOnValues(pass, expr)
		case *ast.StarExpr:
			reportURLDeref(pass, expr)
		}
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// reportMapsCloneOnValues flags maps.Clone applied to url.Values, which leaves
// every value slice shared between the two maps.
func reportMapsCloneOnValues(pass *analysis.Pass, call *ast.CallExpr) {
	if !isCallTo(call, pass.TypesInfo, "maps", "Clone") || len(call.Args) != 1 {
		return
	}

	if !isNamedType(pass.TypesInfo.TypeOf(call.Args[0]), urlPkgPath, urlValuesType) {
		return
	}

	if !goVersionAtLeast(pass, call.Pos(), goVersion127) {
		return
	}

	pass.Reportf(call.Pos(),
		"maps.Clone on url.Values copies the map but shares every []string value; use Values.Clone (Go 1.27)")
}

// reportURLDeref flags *u on a *url.URL, which copies the struct while leaving
// the embedded *Userinfo pointer shared.
//
// A type expression such as the *url.URL in a declaration is also a StarExpr,
// but its operand denotes a type rather than a pointer value, so requiring a
// pointer-typed operand excludes it.
func reportURLDeref(pass *analysis.Pass, star *ast.StarExpr) {
	ptr, isPtr := pass.TypesInfo.TypeOf(star.X).(*types.Pointer)
	if !isPtr {
		return
	}

	if !isNamedType(ptr.Elem(), urlPkgPath, urlURLType) {
		return
	}

	if !goVersionAtLeast(pass, star.Pos(), goVersion127) {
		return
	}

	pass.Reportf(star.Pos(),
		"dereferencing a *url.URL copies the struct but shares the *Userinfo; use URL.Clone (Go 1.27)")
}

// isNamedType reports whether typ is the named type pkgPath.name.
func isNamedType(typ types.Type, pkgPath, name string) bool {
	if typ == nil {
		return false
	}

	named, isNamed := typ.(*types.Named)
	if !isNamed {
		return false
	}

	obj := named.Obj()

	return obj.Pkg() != nil && obj.Pkg().Path() == pkgPath && obj.Name() == name
}
