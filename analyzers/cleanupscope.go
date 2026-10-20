package analyzers

import (
	"go/ast"
	"go/types"
	"sync"

	"golang.org/x/tools/go/analysis"
)

// cleanupScope is the set of function bodies in one package that run as
// testing Cleanup callbacks, plus everything they call. t.Context() is
// canceled before Cleanup runs, so advice to use it must stay out of these
// bodies. A function also called earlier in the test still belongs, because a
// t.Context() there would be dead on the Cleanup path.
type cleanupScope struct {
	lits  map[*ast.FuncLit]bool
	funcs map[*types.Func]bool
}

// cleanupScopeCache memoizes one cleanupScope per package, because the three
// analyzers that suggest t.Context() each run on every package and the scope
// costs a walk of every file.
//
//nolint:gochecknoglobals // process-wide cache shared by every t.Context() analyzer
var cleanupScopeCache sync.Map

// cleanupScopeFor returns the package's cleanupScope, building it once.
func cleanupScopeFor(pass *analysis.Pass) *cleanupScope {
	if cached, ok := cleanupScopeCache.Load(pass.Pkg); ok {
		if scope, ok := cached.(*cleanupScope); ok {
			return scope
		}
	}

	scope := buildCleanupScope(pass)
	cleanupScopeCache.Store(pass.Pkg, scope)

	return scope
}

// contains reports whether the node on top of stack sits inside a Cleanup
// body.
func (scope *cleanupScope) contains(stack []ast.Node, info *types.Info) bool {
	for _, node := range stack {
		switch funcNode := node.(type) {
		case *ast.FuncLit:
			if scope.lits[funcNode] {
				return true
			}
		case *ast.FuncDecl:
			if obj, isFunc := info.Defs[funcNode.Name].(*types.Func); isFunc && scope.funcs[obj] {
				return true
			}
		}
	}

	return false
}

// cleanupBuilder carries the lookups buildCleanupScope needs while it walks
// the package.
type cleanupBuilder struct {
	scope     *cleanupScope
	info      *types.Info
	decls     map[*types.Func]*ast.FuncDecl
	litsByVar map[*types.Var]*ast.FuncLit
}

func buildCleanupScope(pass *analysis.Pass) *cleanupScope {
	builder := &cleanupBuilder{
		scope:     &cleanupScope{lits: map[*ast.FuncLit]bool{}, funcs: map[*types.Func]bool{}},
		info:      pass.TypesInfo,
		decls:     map[*types.Func]*ast.FuncDecl{},
		litsByVar: map[*types.Var]*ast.FuncLit{},
	}

	for _, file := range pass.Files {
		builder.indexFile(file)
	}

	for _, file := range pass.Files {
		ast.Inspect(file, builder.visitCleanupCall)
	}

	return builder.scope
}

// indexFile records every function declaration by its object and every
// function literal bound to a variable, so a Cleanup argument given by name
// can be followed to its body.
func (builder *cleanupBuilder) indexFile(file *ast.File) {
	ast.Inspect(file, func(node ast.Node) bool {
		switch stmt := node.(type) {
		case *ast.FuncDecl:
			builder.indexDecl(stmt)
		case *ast.AssignStmt:
			builder.indexBindings(stmt.Lhs, stmt.Rhs)
		case *ast.ValueSpec:
			builder.indexBindings(identsAsExprs(stmt.Names), stmt.Values)
		}

		return true
	})
}

func (builder *cleanupBuilder) indexDecl(decl *ast.FuncDecl) {
	if obj, isFunc := builder.info.Defs[decl.Name].(*types.Func); isFunc {
		builder.decls[obj] = decl
	}
}

// indexBindings pairs each left-hand identifier with a function literal on
// the right, for both `f := func() {...}` and `var f = func() {...}`.
func (builder *cleanupBuilder) indexBindings(lhs, rhs []ast.Expr) {
	for idx, target := range lhs {
		if idx >= len(rhs) {
			return
		}

		lit, isLit := rhs[idx].(*ast.FuncLit)
		ident, isIdent := target.(*ast.Ident)

		if !isLit || !isIdent {
			continue
		}

		if obj, isVar := builder.info.ObjectOf(ident).(*types.Var); isVar {
			builder.litsByVar[obj] = lit
		}
	}
}

// visitCleanupCall adds the callback of every Cleanup call to the scope.
func (builder *cleanupBuilder) visitCleanupCall(node ast.Node) bool {
	call, isCall := node.(*ast.CallExpr)
	if !isCall || len(call.Args) != 1 || !isTestingCleanupCall(call, builder.info) {
		return true
	}

	builder.addCallback(call.Args[0])

	return true
}

// addCallback follows a Cleanup argument to the body it names.
func (builder *cleanupBuilder) addCallback(arg ast.Expr) {
	if lit, isLit := arg.(*ast.FuncLit); isLit {
		builder.addLit(lit)

		return
	}

	switch obj := calleeObject(arg, builder.info).(type) {
	case *types.Func:
		builder.addFunc(obj)
	case *types.Var:
		if lit := builder.litsByVar[obj]; lit != nil {
			builder.addLit(lit)
		}
	}
}

func (builder *cleanupBuilder) addLit(lit *ast.FuncLit) {
	if builder.scope.lits[lit] {
		return
	}

	builder.scope.lits[lit] = true
	builder.addCallees(lit.Body)
}

func (builder *cleanupBuilder) addFunc(funcObj *types.Func) {
	if builder.scope.funcs[funcObj] {
		return
	}

	builder.scope.funcs[funcObj] = true

	if decl := builder.decls[funcObj]; decl != nil && decl.Body != nil {
		builder.addCallees(decl.Body)
	}
}

// addCallees adds every function or method called from body, since those
// run on the Cleanup path too.
func (builder *cleanupBuilder) addCallees(body *ast.BlockStmt) {
	ast.Inspect(body, func(node ast.Node) bool {
		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}

		if callee, isFunc := calleeObject(call.Fun, builder.info).(*types.Func); isFunc {
			builder.addFunc(callee)
		}

		return true
	})
}

// calleeObject resolves a plain identifier or a selector to the object it
// names, or nil for any other expression.
func calleeObject(expr ast.Expr, info *types.Info) types.Object {
	switch fun := expr.(type) {
	case *ast.Ident:
		return info.ObjectOf(fun)
	case *ast.SelectorExpr:
		return info.ObjectOf(fun.Sel)
	default:
		return nil
	}
}

// isTestingCleanupCall reports whether call is Cleanup on a testing.T, B, F,
// or TB value.
func isTestingCleanupCall(call *ast.CallExpr, info *types.Info) bool {
	sel, isSel := call.Fun.(*ast.SelectorExpr)

	return isSel && sel.Sel.Name == "Cleanup" && isTestingType(info.TypeOf(sel.X))
}

// identsAsExprs widens a ValueSpec name list to the expression list that
// indexBindings takes.
func identsAsExprs(names []*ast.Ident) []ast.Expr {
	exprs := make([]ast.Expr, len(names))
	for idx, name := range names {
		exprs[idx] = name
	}

	return exprs
}
