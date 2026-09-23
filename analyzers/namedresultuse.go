package analyzers

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// namedResultUseAnalyzer returns an analyzer that flags functions whose body
// treats a named result as a working variable, either by assigning it or by
// returning with a naked return. Names that only document the signature, the
// way io.Reader spells Read(p []byte) (n int, err error), are left alone. So
// is any function where a deferred closure assigns a result, because that is
// the only way to propagate a Close error or a recovered panic to the caller.
func namedResultUseAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "namedresultuse",
		Doc:      "flags named results assigned in the body or returned bare; keep result names for documentation and defer",
		Run:      runNamedResultUse,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runNamedResultUse(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		funcType, body, label := funcParts(node)
		if body == nil {
			return
		}

		results := namedResults(pass.TypesInfo, funcType)
		if len(results) == 0 {
			return
		}

		checkNamedResultUse(pass, body, label, results)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// funcParts returns the signature, body, and a human-readable label for a
// FuncDecl or FuncLit. body is nil for declarations without one.
func funcParts(node ast.Node) (*ast.FuncType, *ast.BlockStmt, string) {
	switch fn := node.(type) {
	case *ast.FuncDecl:
		return fn.Type, fn.Body, fn.Name.Name
	case *ast.FuncLit:
		return fn.Type, fn.Body, "func literal"
	default:
		return nil, nil, ""
	}
}

// namedResults returns the objects behind every non-blank result name in the
// signature, keyed so that later identifier lookups can match by identity.
func namedResults(info *types.Info, funcType *ast.FuncType) map[*types.Var]*ast.Ident {
	if funcType.Results == nil {
		return nil
	}

	results := make(map[*types.Var]*ast.Ident, len(funcType.Results.List))

	for _, field := range funcType.Results.List {
		for _, name := range field.Names {
			obj, isVar := info.Defs[name].(*types.Var)
			if !isVar || name.Name == "_" {
				continue
			}

			results[obj] = name
		}
	}

	return results
}

// checkNamedResultUse reports the first naked return and the first assignment
// to each named result in body. Nested function literals are walked for
// assignments, since they can reach the enclosing results, but their own
// return statements belong to them and are skipped.
func checkNamedResultUse(pass *analysis.Pass, body *ast.BlockStmt, label string, results map[*types.Var]*ast.Ident) {
	if assignsResultInDefer(pass.TypesInfo, body, results) {
		return
	}

	reported := make(map[*types.Var]bool, len(results))

	var nakedReported bool

	walkFuncBody(body, func(node ast.Node, inNestedLit bool) {
		if ret, isReturn := node.(*ast.ReturnStmt); isReturn && !inNestedLit && len(ret.Results) == 0 && !nakedReported {
			nakedReported = true

			pass.Reportf(ret.Pos(), "naked return in %s; name results only to document them or to set them in a defer", label)

			return
		}

		for _, target := range assignedIdents(node) {
			obj, isVar := pass.TypesInfo.Uses[target].(*types.Var)
			if !isVar || reported[obj] {
				continue
			}

			resultName, isResult := results[obj]
			if !isResult {
				continue
			}

			reported[obj] = true

			pass.Reportf(target.Pos(), "named result %q is assigned in %s; use a local and return it explicitly", resultName.Name, label)
		}
	})
}

// assignsResultInDefer reports whether any deferred function literal in body
// assigns one of the named results. That shape is the one legitimate reason to
// use a result name as a variable, so it exempts the whole function.
func assignsResultInDefer(info *types.Info, body *ast.BlockStmt, results map[*types.Var]*ast.Ident) bool {
	var found bool

	ast.Inspect(body, func(node ast.Node) bool {
		deferStmt, isDefer := node.(*ast.DeferStmt)
		if !isDefer || found {
			return !found
		}

		lit, isLit := deferStmt.Call.Fun.(*ast.FuncLit)
		if !isLit {
			return true
		}

		found = litAssignsResult(info, lit, results)

		return !found
	})

	return found
}

// litAssignsResult reports whether lit assigns any of results anywhere in its
// body, including through nested closures.
func litAssignsResult(info *types.Info, lit *ast.FuncLit, results map[*types.Var]*ast.Ident) bool {
	var found bool

	ast.Inspect(lit.Body, func(node ast.Node) bool {
		for _, target := range assignedIdents(node) {
			obj, isVar := info.Uses[target].(*types.Var)
			if isVar && results[obj] != nil {
				found = true
			}
		}

		return !found
	})

	return found
}

// assignedIdents returns the identifiers a statement writes to through =,
// op-assignment, ++/--, or a range clause with =. Declarations with := create
// new variables and never touch a named result, so they are not included.
func assignedIdents(node ast.Node) []*ast.Ident {
	switch stmt := node.(type) {
	case *ast.AssignStmt:
		if stmt.Tok == token.DEFINE {
			return nil
		}

		return identsOf(stmt.Lhs)
	case *ast.IncDecStmt:
		return identsOf([]ast.Expr{stmt.X})
	case *ast.RangeStmt:
		if stmt.Tok != token.ASSIGN {
			return nil
		}

		return identsOf([]ast.Expr{stmt.Key, stmt.Value})
	default:
		return nil
	}
}

// identsOf keeps the bare identifiers from exprs. Selector and index targets
// such as n.field = x mutate what the result points at, not the result itself.
func identsOf(exprs []ast.Expr) []*ast.Ident {
	idents := make([]*ast.Ident, 0, len(exprs))

	for _, expr := range exprs {
		if ident, isIdent := expr.(*ast.Ident); isIdent {
			idents = append(idents, ident)
		}
	}

	return idents
}

// walkFuncBody visits every node in body, telling visit whether the node sits
// inside a nested function literal. Each literal is entered exactly once.
func walkFuncBody(body ast.Node, visit func(node ast.Node, inNestedLit bool)) {
	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil {
			return false
		}

		if lit, isLit := node.(*ast.FuncLit); isLit {
			walkNestedLit(lit.Body, visit)

			return false
		}

		visit(node, false)

		return true
	})
}

// walkNestedLit visits the body of a nested function literal with inNestedLit
// set, so callers can ignore its return statements while still seeing the
// assignments it makes to the enclosing function's results.
func walkNestedLit(body ast.Node, visit func(node ast.Node, inNestedLit bool)) {
	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil {
			return false
		}

		visit(node, true)

		return true
	})
}
