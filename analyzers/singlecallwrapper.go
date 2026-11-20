package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// singleCallWrapperAnalyzer returns an analyzer that flags an unexported
// function whose whole body returns one call to another package's function,
// passing its own parameters unchanged beside constants, nil, and
// package-level names. It removes no logic, so the call belongs inline at
// each caller. Each check below takes the narrower reading of an open edge,
// and _test.go files are read too.
func singleCallWrapperAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "singlecallwrapper",
		Doc:      "flags an unexported function whose whole body returns one call to another package's function with its own parameters unchanged and every other argument fixed; call the function directly",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      runSingleCallWrapper,
	}
}

// target is the callee as the wrapper's file spells it, so an import alias
// in the diagnostic matches what the caller would type.
type singleCallWrapper struct {
	name   *ast.Ident
	target string
}

func runSingleCallWrapper(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	wrappers := make(map[*types.Func]singleCallWrapper)

	var order []*types.Func

	insp.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(node ast.Node) {
		funcDecl, isFunc := node.(*ast.FuncDecl)
		if !isFunc {
			return
		}

		if obj, wrapper, found := asSingleCallWrapper(pass, funcDecl); found {
			wrappers[obj] = wrapper
			order = append(order, obj)
		}
	})

	dropValueUses(pass, insp, wrappers)

	otherBuilds := ignoredFuncNames(pass)

	for _, obj := range order {
		wrapper, kept := wrappers[obj]
		if !kept || otherBuilds[obj.Name()] {
			continue
		}

		pass.Reportf(wrapper.name.Pos(), "function %s only returns %s called with its own parameters and fixed arguments; call %s directly at each caller",
			wrapper.name.Name, wrapper.target, wrapper.target)
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

func asSingleCallWrapper(pass *analysis.Pass, funcDecl *ast.FuncDecl) (*types.Func, singleCallWrapper, bool) {
	call := returnedCall(funcDecl)
	if call == nil {
		return nil, singleCallWrapper{}, false
	}

	obj, isFunc := pass.TypesInfo.Defs[funcDecl.Name].(*types.Func)
	if !isFunc {
		return nil, singleCallWrapper{}, false
	}

	if foreignFunc(pass, call) == nil {
		return nil, singleCallWrapper{}, false
	}

	sig := obj.Signature()
	if !forwardsParams(pass, sig, call) || !sameResults(pass, sig, call) {
		return nil, singleCallWrapper{}, false
	}

	return obj, singleCallWrapper{name: funcDecl.Name, target: types.ExprString(ast.Unparen(call.Fun))}, true
}

// An exported function or a //go:linkname target may have callers outside
// the package that the pass cannot see, a method may satisfy an interface,
// and a generic function cannot always hand its type parameters through, so
// all of them are left to the reviewer.
func returnedCall(funcDecl *ast.FuncDecl) *ast.CallExpr {
	if funcDecl.Recv != nil || funcDecl.Name.IsExported() || funcDecl.Type.TypeParams != nil || funcDecl.Body == nil {
		return nil
	}

	if hasLinkname(funcDecl.Doc) {
		return nil
	}

	if len(funcDecl.Body.List) != 1 {
		return nil
	}

	ret, isReturn := funcDecl.Body.List[0].(*ast.ReturnStmt)
	if !isReturn || len(ret.Results) != 1 {
		return nil
	}

	call, isCall := ast.Unparen(ret.Results[0]).(*ast.CallExpr)
	if !isCall {
		return nil
	}

	return call
}

// A builtin, a conversion, a call through a func value, a method, and a
// function of the same package all return nil: none is a function another
// package declares.
func foreignFunc(pass *analysis.Pass, call *ast.CallExpr) *types.Func {
	callee := typeutil.StaticCallee(pass.TypesInfo, call)
	if callee == nil || callee.Signature().Recv() != nil {
		return nil
	}

	if callee.Pkg() == nil || callee.Pkg() == pass.Pkg {
		return nil
	}

	return callee
}

// Every parameter must appear once, as itself, in declaration order. A
// dropped, blank, repeated, transformed, or reordered parameter carries a
// choice of its own, as a reversed comparator does, so it is not reported.
// A function with no parameters names a fixed call rather than wrapping
// the caller's values, so it is skipped too.
func forwardsParams(pass *analysis.Pass, sig *types.Signature, call *ast.CallExpr) bool {
	params := sig.Params()
	if params.Len() == 0 {
		return false
	}

	var next int

	for _, arg := range call.Args {
		index, isParam := paramIndex(pass, params, arg)
		if !isParam && !isFixedArg(pass, arg) {
			return false
		}

		if !isParam {
			continue
		}

		if index != next {
			return false
		}

		next++
	}

	return next == params.Len() && forwardsVariadic(sig, call)
}

// A variadic parameter must reach the call through `...`; passing it as one
// slice changes how the callee receives it.
func forwardsVariadic(sig *types.Signature, call *ast.CallExpr) bool {
	if !sig.Variadic() {
		return true
	}

	return call.Ellipsis != token.NoPos
}

func paramIndex(pass *analysis.Pass, params *types.Tuple, arg ast.Expr) (int, bool) {
	ident, isIdent := ast.Unparen(arg).(*ast.Ident)
	if !isIdent {
		return 0, false
	}

	obj := pass.TypesInfo.Uses[ident]

	for index := range params.Len() {
		if params.At(index) == obj {
			return index, true
		}
	}

	return 0, false
}

// A local, a field, a call result, or a composite literal can vary or hold
// a rule of its own, so only constants, nil, and package-level names count.
func isFixedArg(pass *analysis.Pass, arg ast.Expr) bool {
	value := pass.TypesInfo.Types[arg]
	if value.Value != nil || value.IsNil() {
		return true
	}

	ident := identOf(ast.Unparen(arg))
	if ident == nil {
		return false
	}

	obj := pass.TypesInfo.Uses[ident]
	if obj == nil || obj.Pkg() == nil {
		return false
	}

	return obj.Parent() == obj.Pkg().Scope()
}

// A wrapper that changes the result type, such as returning an interface
// in place of the callee's concrete type, hides that type on purpose.
func sameResults(pass *analysis.Pass, sig *types.Signature, call *ast.CallExpr) bool {
	results := sig.Results()
	callType := pass.TypesInfo.TypeOf(call)

	tuple, isTuple := callType.(*types.Tuple)
	if !isTuple {
		return results.Len() == 1 && types.Identical(results.At(0).Type(), callType)
	}

	for index := range tuple.Len() {
		if !types.Identical(tuple.At(index).Type(), results.At(index).Type()) {
			return false
		}
	}

	return true
}

// A wrapper used as a func value, not only called, stands in where the
// callee's own signature does not fit, so it cannot be inlined.
func dropValueUses(pass *analysis.Pass, insp *inspector.Inspector, wrappers map[*types.Func]singleCallWrapper) {
	if len(wrappers) == 0 {
		return
	}

	called := make(map[*ast.Ident]bool)

	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(node ast.Node) {
		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return
		}

		if ident := identOf(ast.Unparen(call.Fun)); ident != nil {
			called[ident] = true
		}
	})

	for ident, obj := range pass.TypesInfo.Uses {
		funcDecl, isFunc := obj.(*types.Func)
		if isFunc && !called[ident] {
			delete(wrappers, funcDecl)
		}
	}
}

func hasLinkname(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}

	for _, comment := range doc.List {
		if strings.HasPrefix(comment.Text, "//go:linkname ") {
			return true
		}
	}

	return false
}

// A function a build constraint swaps per platform picks a different callee
// or result type in each build, so callers cannot inline it. A file of
// another package, such as a `//go:build ignore` generator in package main,
// swaps nothing.
func ignoredFuncNames(pass *analysis.Pass) map[string]bool {
	names := make(map[string]bool)

	for _, path := range pass.IgnoredFiles {
		for _, name := range topLevelNames(path, pass.Pkg.Name()) {
			names[name] = true
		}
	}

	return names
}

// A file that fails to parse names nothing, so its wrappers stay reported.
func topLevelNames(path, pkgName string) []string {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil || file.Name.Name != pkgName {
		return nil
	}

	var names []string

	for _, decl := range file.Decls {
		names = append(names, declNames(decl)...)
	}

	return names
}

// Another build may declare the name as a package-level variable holding a
// func, such as `var openFile = os.Create`, instead of as a function.
func declNames(decl ast.Decl) []string {
	switch typed := decl.(type) {
	case *ast.FuncDecl:
		if typed.Recv == nil {
			return []string{typed.Name.Name}
		}
	case *ast.GenDecl:
		if typed.Tok == token.VAR {
			return valueSpecNames(typed.Specs)
		}
	}

	return nil
}

func valueSpecNames(specs []ast.Spec) []string {
	var names []string

	for _, spec := range specs {
		valueSpec, isValue := spec.(*ast.ValueSpec)
		if !isValue {
			continue
		}

		for _, name := range valueSpec.Names {
			names = append(names, name.Name)
		}
	}

	return names
}
