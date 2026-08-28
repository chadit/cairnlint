package analyzers

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

type passthroughMethod struct {
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

// agentPassthroughAnalyzer flags methods that only forward their arguments to
// another value, and types whose every method does so. A forwarding method
// adds a name and a call hop; the caller could hold the wrapped value instead.
// Fowler calls the type-level shape a Middle Man.
//
// It is agent-only because thin forwarding is sometimes the point: an adapter
// at a package boundary, a method promoted to satisfy an external interface, or
// a facade that deliberately narrows a wide type. The analyzer cannot see the
// caller's reason, so it reports the shape and an LLM decides.
func agentPassthroughAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "agentpassthrough",
		Doc:  "[agent] flags forward-only methods and types whose every method forwards (middle man)",
		Run:  runAgentPassthrough,
	}
}

func runAgentPassthrough(pass *analysis.Pass) (any, error) {
	byReceiver := make(map[string]*receiverMethods)
	typePos := make(map[string]token.Pos)

	for _, file := range pass.Files {
		collectPassthrough(pass, file, byReceiver, typePos)
	}

	for recv, methods := range byReceiver {
		reportPassthrough(pass, recv, methods, typePos)
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

func collectPassthrough(pass *analysis.Pass, file *ast.File, byReceiver map[string]*receiverMethods, typePos map[string]token.Pos) {
	if isTestFile(pass, file) {
		return
	}

	for _, decl := range file.Decls {
		recordTypePositions(decl, typePos)
		recordMethod(decl, byReceiver)
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

func recordMethod(decl ast.Decl, byReceiver map[string]*receiverMethods) {
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

	target, forwards := forwardTarget(funcDecl)
	if forwards {
		methods.forward = append(methods.forward, passthroughMethod{name: funcDecl.Name.Name, target: target, pos: funcDecl.Name.Pos()})
	}
}

// reportPassthrough emits the type-level finding when every method forwards
// out of a single field, and the per-method finding otherwise. Reporting one
// or the other keeps a wrapper with eight methods from producing eight
// diagnostics that all say the same thing.
func reportPassthrough(pass *analysis.Pass, recv string, methods *receiverMethods, typePos map[string]token.Pos) {
	if len(methods.forward) == 0 {
		return
	}

	pos, known := typePos[recv]
	if known && methods.total == len(methods.forward) && singleFieldStruct(pass, recv) {
		pass.Reportf(pos, "type %s wraps one field and every one of its %d methods forwards; callers can hold the wrapped value directly", recv, methods.total)

		return
	}

	for _, method := range methods.forward {
		pass.Reportf(method.pos, "method %s.%s only forwards its arguments to %s and adds no behavior of its own", recv, method.name, method.target)
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
func forwardTarget(funcDecl *ast.FuncDecl) (string, bool) {
	call, isSingleCall := soleCall(funcDecl.Body)
	if !isSingleCall {
		return "", false
	}

	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel {
		return "", false
	}

	if !forwardsAllParams(funcDecl, call) {
		return "", false
	}

	return renderSelector(sel), true
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
