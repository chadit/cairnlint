package analyzers

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"iter"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// net/http names status codes only from 100 through 599, so a literal
// outside that range never has a constant to suggest.
const (
	minHTTPStatus = 100
	maxHTTPStatus = 599
)

// Argument positions of the status code in http.Error and http.Redirect.
const (
	httpErrorStatusArg    = 2
	httpRedirectStatusArg = 3
)

// Names looked up in net/http.
const (
	httpStatusPrefix   = "Status"
	responseType       = "Response"
	responseWriterType = "ResponseWriter"
	statusCodeField    = "StatusCode"
	writeHeaderMethod  = "WriteHeader"
	redirectFunc       = "Redirect"
)

// httpStatusSink marks a field or variable its declaring package hands to
// net/http as a response status. It travels as a fact because the literal
// that should name the status usually sits in another package, as when a
// fake server's struct is filled in by a test elsewhere.
type httpStatusSink struct{}

func (*httpStatusSink) AFact() {}

func (*httpStatusSink) String() string { return "httpStatusSink" }

// httpStatusLiteralAnalyzer returns an analyzer that flags a status literal
// set on, assigned to, or compared with a field or variable that reaches a
// response status. Where the value goes is the signal: usestdlibvars sees
// only net/http's own fields, and a name match on Status would also catch
// SMTP replies and exit codes.
func httpStatusLiteralAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:      "httpstatusliteral",
		Doc:       "flags an int literal net/http names as a Status constant when it is written to or compared with a field or variable passed to WriteHeader, http.Error, http.Redirect, or http.Response.StatusCode",
		Requires:  []*analysis.Analyzer{inspect.Analyzer},
		Run:       runHTTPStatusLiteral,
		FactTypes: []analysis.Fact{new(httpStatusSink)},
	}
}

func runHTTPStatusLiteral(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	checker := newStatusChecker(pass)
	if checker == nil {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	// Every mark in the package has to exist before any literal is judged,
	// since a write can come before the WriteHeader call that marks its target.
	insp.Preorder([]ast.Node{
		(*ast.CallExpr)(nil),
		(*ast.AssignStmt)(nil),
		(*ast.CompositeLit)(nil),
	}, checker.markSinks)

	insp.Preorder([]ast.Node{
		(*ast.CompositeLit)(nil),
		(*ast.AssignStmt)(nil),
		(*ast.ValueSpec)(nil),
		(*ast.BinaryExpr)(nil),
		(*ast.SwitchStmt)(nil),
	}, checker.reportLiterals)

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// statusChecker holds what both walks over one package share: the net/http
// types a status reaches, and the package's own marked objects. Locals and
// parameters are marked here only, since no other package can name them.
type statusChecker struct {
	pass           *analysis.Pass
	httpPkg        *types.Package
	responseWriter *types.Interface
	responseStatus *types.Var
	marked         map[*types.Var]bool
}

// newStatusChecker returns nil when the package cannot reach net/http, as
// then nothing it touches can carry a response status.
func newStatusChecker(pass *analysis.Pass) *statusChecker {
	httpPkg := importedPackage(pass.Pkg, httpPkgPath)
	if httpPkg == nil {
		return nil
	}

	writer, isIface := lookupType(httpPkg, responseWriterType).Underlying().(*types.Interface)
	if !isIface {
		return nil
	}

	statusObj, _, _ := types.LookupFieldOrMethod(lookupType(httpPkg, responseType), false, httpPkg, statusCodeField)

	status, isVar := statusObj.(*types.Var)
	if !isVar {
		return nil
	}

	return &statusChecker{
		pass:           pass,
		httpPkg:        httpPkg,
		responseWriter: writer,
		responseStatus: status,
		marked:         make(map[*types.Var]bool),
	}
}

// importedPackage finds path in the import graph of root, root included. The
// search is transitive because a test that imports only a fake server still
// writes a status that reaches net/http through it.
func importedPackage(root *types.Package, path string) *types.Package {
	seen := map[*types.Package]bool{root: true}
	queue := []*types.Package{root}

	for len(queue) > 0 {
		pkg := queue[0]
		queue = queue[1:]

		if pkg.Path() == path {
			return pkg
		}

		queue = enqueueUnseen(queue, seen, pkg.Imports())
	}

	return nil
}

func enqueueUnseen(queue []*types.Package, seen map[*types.Package]bool, pkgs []*types.Package) []*types.Package {
	for _, pkg := range pkgs {
		if seen[pkg] {
			continue
		}

		seen[pkg] = true
		queue = append(queue, pkg)
	}

	return queue
}

// lookupType returns the type pkg declares under name, or the invalid type
// when it declares none, so a caller's Underlying assertion fails cleanly.
func lookupType(pkg *types.Package, name string) types.Type {
	typeName, isType := pkg.Scope().Lookup(name).(*types.TypeName)
	if !isType {
		return types.Typ[types.Invalid]
	}

	return typeName.Type()
}

// markSinks records each field or variable node hands straight to net/http
// as a response status.
func (sc *statusChecker) markSinks(node ast.Node) {
	switch typed := node.(type) {
	case *ast.CallExpr:
		sc.mark(sc.statusArg(typed))
	case *ast.AssignStmt:
		sc.markResponseAssign(typed)
	case *ast.CompositeLit:
		sc.markResponseLiteral(typed)
	}
}

func (sc *statusChecker) markResponseAssign(assign *ast.AssignStmt) {
	for target, value := range assignPairs(assign) {
		if sc.varOf(target) == sc.responseStatus {
			sc.mark(value)
		}
	}
}

func (sc *statusChecker) markResponseLiteral(lit *ast.CompositeLit) {
	for field, value := range structLitFields(sc.pass.TypesInfo, lit) {
		if field == sc.responseStatus {
			sc.mark(value)
		}
	}
}

// statusArg returns the argument call passes as a response status, or nil
// when call is not WriteHeader on a ResponseWriter, http.Error, or
// http.Redirect. A method expression call takes its receiver as the first
// argument, so its arguments do not line up with the parameters and it is
// skipped.
func (sc *statusChecker) statusArg(call *ast.CallExpr) ast.Expr {
	callee, isFunc := typeutil.Callee(sc.pass.TypesInfo, call).(*types.Func)
	if !isFunc || len(call.Args) != callee.Signature().Params().Len() {
		return nil
	}

	argIdx, isSink := sc.statusArgIndex(callee)
	if !isSink {
		return nil
	}

	return call.Args[argIdx]
}

// statusArgIndex reports where callee takes a response status. Any method
// named WriteHeader counts when its receiver implements http.ResponseWriter,
// which covers wrappers and recorders as well as the interface itself.
func (sc *statusChecker) statusArgIndex(callee *types.Func) (int, bool) {
	if recv := callee.Signature().Recv(); recv != nil {
		return 0, callee.Name() == writeHeaderMethod && types.Implements(recv.Type(), sc.responseWriter)
	}

	if callee.Pkg() == nil || callee.Pkg().Path() != httpPkgPath {
		return 0, false
	}

	switch callee.Name() {
	case errorName:
		return httpErrorStatusArg, true
	case redirectFunc:
		return httpRedirectStatusArg, true
	}

	return 0, false
}

// mark records the field or variable expr names when this package declares
// it, and exports a fact for it when another package can name it. Only the
// declaring package marks an object, as only it can export the fact.
func (sc *statusChecker) mark(expr ast.Expr) {
	target := sc.varOf(expr)
	if target == nil || target.Pkg() != sc.pass.Pkg {
		return
	}

	sc.marked[target] = true

	if target.IsField() || target.Parent() == target.Pkg().Scope() {
		sc.pass.ExportObjectFact(target, new(httpStatusSink))
	}
}

// varOf returns the field or variable expr names directly, resolved to its
// generic origin so every instantiation shares one mark. Anything else,
// including a call or a conversion, yields nil: only a direct pass counts.
func (sc *statusChecker) varOf(expr ast.Expr) *types.Var {
	var ident *ast.Ident

	switch typed := ast.Unparen(expr).(type) {
	case *ast.Ident:
		ident = typed
	case *ast.SelectorExpr:
		ident = typed.Sel
	default:
		return nil
	}

	variable, isVar := sc.pass.TypesInfo.ObjectOf(ident).(*types.Var)
	if !isVar {
		return nil
	}

	return variable.Origin()
}

// isMarked reports whether target reaches a response status, through this
// package's own walk when it declares target and through the declaring
// package's fact otherwise.
func (sc *statusChecker) isMarked(target *types.Var) bool {
	if target == nil {
		return false
	}

	if target.Pkg() == sc.pass.Pkg {
		return sc.marked[target]
	}

	return sc.pass.ImportObjectFact(target, new(httpStatusSink))
}

// reportLiterals flags each status literal node writes to or compares with a
// marked field or variable. A const declaration never reaches a report,
// because a constant is not a variable and so is never marked.
func (sc *statusChecker) reportLiterals(node ast.Node) {
	switch typed := node.(type) {
	case *ast.CompositeLit:
		sc.reportLiteralFields(typed)
	case *ast.AssignStmt:
		sc.reportAssignedLiterals(typed)
	case *ast.ValueSpec:
		sc.reportInitLiterals(typed)
	case *ast.BinaryExpr:
		sc.reportComparedLiteral(typed)
	case *ast.SwitchStmt:
		sc.reportCaseLiterals(typed)
	}
}

func (sc *statusChecker) reportLiteralFields(lit *ast.CompositeLit) {
	for field, value := range structLitFields(sc.pass.TypesInfo, lit) {
		if sc.isMarked(field) {
			sc.reportLiteral(value)
		}
	}
}

func (sc *statusChecker) reportAssignedLiterals(assign *ast.AssignStmt) {
	for target, value := range assignPairs(assign) {
		if sc.isMarked(sc.varOf(target)) {
			sc.reportLiteral(value)
		}
	}
}

func (sc *statusChecker) reportInitLiterals(spec *ast.ValueSpec) {
	if len(spec.Values) != len(spec.Names) {
		return
	}

	for idx, name := range spec.Names {
		if sc.isMarked(sc.varOf(name)) {
			sc.reportLiteral(spec.Values[idx])
		}
	}
}

// reportComparedLiteral covers == and != only. An ordering such as
// code >= 500 bounds a class of statuses, which no single constant names.
func (sc *statusChecker) reportComparedLiteral(binary *ast.BinaryExpr) {
	if binary.Op != token.EQL && binary.Op != token.NEQ {
		return
	}

	if sc.isMarked(sc.varOf(binary.X)) {
		sc.reportLiteral(binary.Y)
	}

	if sc.isMarked(sc.varOf(binary.Y)) {
		sc.reportLiteral(binary.X)
	}
}

func (sc *statusChecker) reportCaseLiterals(switchStmt *ast.SwitchStmt) {
	if switchStmt.Tag == nil || !sc.isMarked(sc.varOf(switchStmt.Tag)) {
		return
	}

	for _, stmt := range switchStmt.Body.List {
		clause, isClause := stmt.(*ast.CaseClause)
		if !isClause {
			continue
		}

		for _, expr := range clause.List {
			sc.reportLiteral(expr)
		}
	}
}

// reportLiteral flags expr when it is an int literal net/http names.
func (sc *statusChecker) reportLiteral(expr ast.Expr) {
	lit, isLit := ast.Unparen(expr).(*ast.BasicLit)
	if !isLit || lit.Kind != token.INT {
		return
	}

	name, isNamed := sc.statusName(sc.pass.TypesInfo.Types[lit].Value)
	if !isNamed {
		return
	}

	sc.pass.Reportf(lit.Pos(), "literal HTTP status %s; use http.%s", lit.Value, name)
}

// statusName returns the net/http constant whose value is code. It reads
// net/http's own declarations rather than a copied table, so the set is
// exactly what the checked code compiles against: 306, which net/http
// leaves unnamed, and gaps such as 299 never match.
func (sc *statusChecker) statusName(code constant.Value) (string, bool) {
	if code == nil || code.Kind() != constant.Int {
		return "", false
	}

	value, isExact := constant.Int64Val(code)
	if !isExact || value < minHTTPStatus || value > maxHTTPStatus {
		return "", false
	}

	scope := sc.httpPkg.Scope()
	for _, name := range scope.Names() {
		if isStatusConst(scope.Lookup(name), value) {
			return name, true
		}
	}

	return "", false
}

func isStatusConst(obj types.Object, value int64) bool {
	statusConst, isConst := obj.(*types.Const)
	if !isConst || !strings.HasPrefix(statusConst.Name(), httpStatusPrefix) || statusConst.Val().Kind() != constant.Int {
		return false
	}

	got, isExact := constant.Int64Val(statusConst.Val())

	return isExact && got == value
}

// assignPairs yields each target of a plain or defining assignment with the
// value it receives. A tuple from one call has no value per target, so it
// yields nothing.
func assignPairs(assign *ast.AssignStmt) iter.Seq2[ast.Expr, ast.Expr] {
	return func(yield func(ast.Expr, ast.Expr) bool) {
		if (assign.Tok != token.ASSIGN && assign.Tok != token.DEFINE) || len(assign.Lhs) != len(assign.Rhs) {
			return
		}

		for idx, target := range assign.Lhs {
			if !yield(target, assign.Rhs[idx]) {
				return
			}
		}
	}
}

// structLitFields yields each field a struct literal sets with the value it
// sets, the field resolved to its generic origin. A positional literal pairs
// element i with field i.
func structLitFields(info *types.Info, lit *ast.CompositeLit) iter.Seq2[*types.Var, ast.Expr] {
	return func(yield func(*types.Var, ast.Expr) bool) {
		fields := litStruct(info.TypeOf(lit))
		if fields == nil {
			return
		}

		for idx, elt := range lit.Elts {
			field, value := litElement(info, fields, idx, elt)
			if field != nil && !yield(field.Origin(), value) {
				return
			}
		}
	}
}

// litStruct returns the struct type a composite literal builds, or nil for
// any other literal. It looks through the pointer an elided &T leaves on an
// element of a []*T literal.
func litStruct(typ types.Type) *types.Struct {
	if ptr, isPtr := typ.(*types.Pointer); isPtr {
		typ = ptr.Elem()
	}

	if typ == nil {
		return nil
	}

	fields, _ := typ.Underlying().(*types.Struct)

	return fields
}

func litElement(info *types.Info, fields *types.Struct, idx int, elt ast.Expr) (*types.Var, ast.Expr) {
	if keyValue, isKeyed := elt.(*ast.KeyValueExpr); isKeyed {
		key, isIdent := keyValue.Key.(*ast.Ident)
		if !isIdent {
			return nil, nil
		}

		field, _ := info.ObjectOf(key).(*types.Var)

		return field, keyValue.Value
	}

	if idx >= fields.NumFields() {
		return nil, nil
	}

	return fields.Field(idx), elt
}
