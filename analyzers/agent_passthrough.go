package analyzers

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// inner is the type of the value the method forwards to, nil when the call
// goes to a package-level function: a package qualifier has no entry in
// TypesInfo.Types, while TypeOf would give the invalid type.
type passthroughMethod struct {
	inner  types.Type
	name   string
	target string
	pos    token.Pos
}

// Tracking the total alongside the forwarders is what separates a lone
// forwarder on a working type from a type whose whole method set forwards.
type receiverMethods struct {
	forward []passthroughMethod
	total   int
}

// passthroughIndex holds what the pass's non-test files declare, collected
// across files before reporting: each receiver's methods that have a body and
// where each type is declared.
type passthroughIndex struct {
	byReceiver map[string]*receiverMethods
	typePos    map[string]token.Pos
	required   map[string][]*types.Interface
}

// agentPassthroughAnalyzer flags methods that only forward their arguments to
// another value, and types whose every method does so (Fowler's Middle Man),
// skipping forwarders an interface requires. It is agent-only because
// forwarding is still sometimes the point, as in a boundary adapter or a
// facade narrowing a wide type, and only an LLM can weigh the reason.
func agentPassthroughAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "agentpassthrough",
		Doc:  "[agent] flags forward-only methods and types whose every method forwards (middle man)",
		Run:  runAgentPassthrough,
	}
}

func runAgentPassthrough(pass *analysis.Pass) (any, error) {
	index := passthroughIndex{
		byReceiver: make(map[string]*receiverMethods),
		typePos:    make(map[string]token.Pos),
	}

	for _, file := range pass.Files {
		index.collect(pass, file)
	}

	if index.hasForwards() {
		index.required = interfacesByMethod(pass)
	}

	for recv, methods := range index.byReceiver {
		index.report(pass, recv, methods)
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

func (index *passthroughIndex) collect(pass *analysis.Pass, file *ast.File) {
	if isTestFile(pass, file) {
		return
	}

	for _, decl := range file.Decls {
		recordTypePositions(decl, index.typePos)
		recordMethod(pass, decl, index.byReceiver)
	}
}

// A wrapper finding anchors on the type, not on whichever method came first.
func recordTypePositions(decl ast.Decl, typePos map[string]token.Pos) {
	genDecl, isGen := decl.(*ast.GenDecl)
	if !isGen || genDecl.Tok != token.TYPE {
		return
	}

	for _, spec := range genDecl.Specs {
		typeSpec, isType := spec.(*ast.TypeSpec)
		if isType {
			typePos[typeSpec.Name.Name] = typeSpec.Name.Pos()
		}
	}
}

func recordMethod(pass *analysis.Pass, decl ast.Decl, byReceiver map[string]*receiverMethods) {
	funcDecl, isFunc := decl.(*ast.FuncDecl)
	if !isFunc || funcDecl.Recv == nil || funcDecl.Body == nil {
		return
	}

	recv, hasRecv := receiverTypeName(funcDecl)
	if !hasRecv {
		return
	}

	methods, seen := byReceiver[recv]
	if !seen {
		methods = &receiverMethods{}
		byReceiver[recv] = methods
	}

	methods.total++

	sel, forwards := forwardTarget(funcDecl)
	if forwards {
		methods.forward = append(methods.forward, passthroughMethod{
			inner:  pass.TypesInfo.Types[sel.X].Type,
			name:   funcDecl.Name.Name,
			target: renderSelector(sel),
			pos:    funcDecl.Name.Pos(),
		})
	}
}

// forwardsPhrase says that all count methods forward, in the singular for one
// method so the message never reads "its 1 methods".
func forwardsPhrase(count int) string {
	if count == 1 {
		return "its only method forwards"
	}

	return fmt.Sprintf("every one of its %d methods forwards", count)
}

// report emits the type-level finding when every method forwards
// out of a single field, and the per-method finding otherwise. Reporting one
// or the other keeps a wrapper with eight methods from producing eight
// diagnostics that all say the same thing.
func (index *passthroughIndex) report(pass *analysis.Pass, recv string, methods *receiverMethods) {
	forward := index.removableForwards(pass, recv, methods.forward)
	if len(forward) == 0 {
		return
	}

	pos, known := index.typePos[recv]
	if known && methods.total == len(forward) && singleFieldStruct(pass, recv) {
		pass.Reportf(pos, "[agent] type %s wraps one field and %s; callers can hold the wrapped value directly", recv, forwardsPhrase(methods.total))

		return
	}

	for _, method := range forward {
		pass.Reportf(method.pos, "[agent] method %s.%s only forwards its arguments to %s and adds no behavior of its own", recv, method.name, method.target)
	}
}

// singleFieldStruct reports whether recv is a struct with exactly one field.
// A one-field struct whose methods all forward is a wrapper around that field;
// a wider struct is coordinating something and is not the same finding.
func singleFieldStruct(pass *analysis.Pass, recv string) bool {
	if pass.Pkg == nil {
		return false
	}

	obj := pass.Pkg.Scope().Lookup(recv)
	if obj == nil {
		return false
	}

	strct, isStruct := obj.Type().Underlying().(*types.Struct)

	return isStruct && strct.NumFields() == 1
}

func receiverTypeName(funcDecl *ast.FuncDecl) (string, bool) {
	if len(funcDecl.Recv.List) == 0 {
		return "", false
	}

	expr := funcDecl.Recv.List[0].Type
	if star, isStar := expr.(*ast.StarExpr); isStar {
		expr = star.X
	}

	// A generic receiver arrives as Type[T]; the name is the base expression.
	if index, isIndex := expr.(*ast.IndexExpr); isIndex {
		expr = index.X
	}

	ident, isIdent := expr.(*ast.Ident)
	if !isIdent {
		return "", false
	}

	return ident.Name, true
}

// A body that branches, reshapes an argument, or handles an error is doing
// work, so only a bare single call counts as a forward.
func forwardTarget(funcDecl *ast.FuncDecl) (*ast.SelectorExpr, bool) {
	call, isSingleCall := soleCall(funcDecl.Body)
	if !isSingleCall {
		return nil, false
	}

	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel || !forwardsAllParams(funcDecl, call) {
		return nil, false
	}

	return sel, true
}

func soleCall(body *ast.BlockStmt) (*ast.CallExpr, bool) {
	if len(body.List) != 1 {
		return nil, false
	}

	switch stmt := body.List[0].(type) {
	case *ast.ExprStmt:
		call, isCall := stmt.X.(*ast.CallExpr)

		return call, isCall
	case *ast.ReturnStmt:
		if len(stmt.Results) != 1 {
			return nil, false
		}

		call, isCall := stmt.Results[0].(*ast.CallExpr)

		return call, isCall
	}

	return nil, false
}

func forwardsAllParams(funcDecl *ast.FuncDecl, call *ast.CallExpr) bool {
	params, named := paramNames(funcDecl)
	if !named || len(call.Args) != len(params) {
		return false
	}

	for i, arg := range call.Args {
		ident, isIdent := arg.(*ast.Ident)
		if !isIdent || ident.Name != params[i] {
			return false
		}
	}

	return true
}

// A blank or unnamed parameter cannot be forwarded by name, so those report
// false rather than a short list that would match the wrong call.
func paramNames(funcDecl *ast.FuncDecl) ([]string, bool) {
	if funcDecl.Type.Params == nil {
		return nil, true
	}

	names := make([]string, 0, len(funcDecl.Type.Params.List))

	for _, field := range funcDecl.Type.Params.List {
		if len(field.Names) == 0 {
			return nil, false
		}

		for _, ident := range field.Names {
			if ident.Name == "_" {
				return nil, false
			}

			names = append(names, ident.Name)
		}
	}

	return names, true
}

// Anything the walk cannot render falls back to the trailing method name.
func renderSelector(sel *ast.SelectorExpr) string {
	switch base := sel.X.(type) {
	case *ast.Ident:
		return base.Name + "." + sel.Sel.Name
	case *ast.SelectorExpr:
		return renderSelector(base) + "." + sel.Sel.Name
	}

	return sel.Sel.Name
}

// Indexing every imported interface costs more than the rest of the pass, so
// it waits until some method forwards.
func (index *passthroughIndex) hasForwards() bool {
	for _, methods := range index.byReceiver {
		if len(methods.forward) > 0 {
			return true
		}
	}

	return false
}

// A forwarder is not removable when an interface the type implements declares
// that method and the inner value does not implement that interface, because
// the inner value cannot stand in for the wrapper. When the inner value
// implements it too, the wrapper can go and the forward stands.
func (index *passthroughIndex) removableForwards(pass *analysis.Pass, recv string, forward []passthroughMethod) []passthroughMethod {
	wrapperType := pass.Pkg.Scope().Lookup(recv)
	if wrapperType == nil {
		return forward
	}

	var removable []passthroughMethod

	for _, method := range forward {
		if !index.requiredBy(wrapperType.Type(), method) {
			removable = append(removable, method)
		}
	}

	return removable
}

func (index *passthroughIndex) requiredBy(wrapperType types.Type, method passthroughMethod) bool {
	for _, iface := range index.required[method.name] {
		if !implementsByValueOrPointer(wrapperType, iface) {
			continue
		}

		if method.inner == nil || !implementsByValueOrPointer(method.inner, iface) {
			return true
		}
	}

	return false
}

// interfacesByMethod indexes the named interfaces declared in the package and
// in the packages it imports, standard library included, by method name. An
// empty interface, a type-set constraint, and a generic interface require no
// method a concrete type can be checked against, so they are left out.
func interfacesByMethod(pass *analysis.Pass) map[string][]*types.Interface {
	byMethod := make(map[string][]*types.Interface)

	for _, pkg := range append([]*types.Package{pass.Pkg}, pass.Pkg.Imports()...) {
		for _, name := range pkg.Scope().Names() {
			addInterfaceMethods(byMethod, pkg.Scope().Lookup(name))
		}
	}

	return byMethod
}

func addInterfaceMethods(byMethod map[string][]*types.Interface, obj types.Object) {
	typeName, isType := obj.(*types.TypeName)
	if !isType {
		return
	}

	named, isNamed := typeName.Type().(*types.Named)
	if !isNamed || named.TypeParams().Len() > 0 {
		return
	}

	iface, isIface := named.Underlying().(*types.Interface)
	if !isIface || !iface.IsMethodSet() {
		return
	}

	for method := range iface.Methods() {
		byMethod[method.Name()] = append(byMethod[method.Name()], iface)
	}
}
