package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// chanDirCloseAnalyzer returns an analyzer that flags close() calls on
// bidirectional channel parameters. If a function receives a bidirectional
// channel, closing it is a smell because the function may be acting as a
// receiver, and only senders should close channels.
func chanDirCloseAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "chandirclose",
		Doc:      "flags close() on bidirectional channel parameters; only senders should close channels",
		Run:      runChanDirClose,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runChanDirClose(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return
		}

		if !isBuiltinClose(call, pass.TypesInfo) {
			return
		}

		if len(call.Args) == 0 {
			return
		}

		arg := call.Args[0]
		if !isBidirectionalChanParam(arg, pass.TypesInfo) {
			return
		}

		pass.Reportf(call.Pos(), "close() on bidirectional channel parameter; specify chan<- direction if this function owns the channel")
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isBuiltinClose reports whether call is a call to the builtin close().
func isBuiltinClose(call *ast.CallExpr, info *types.Info) bool {
	ident, isIdent := call.Fun.(*ast.Ident)
	if !isIdent || ident.Name != "close" {
		return false
	}

	obj := info.Uses[ident]
	if obj == nil {
		return false
	}

	_, isBuiltin := obj.(*types.Builtin)

	return isBuiltin
}

// isBidirectionalChanParam reports whether expr is an identifier that
// resolves to a function parameter with bidirectional channel type.
func isBidirectionalChanParam(expr ast.Expr, info *types.Info) bool {
	ident, isIdent := expr.(*ast.Ident)
	if !isIdent {
		return false
	}

	obj := info.ObjectOf(ident)
	if obj == nil {
		return false
	}

	varObj, isVar := obj.(*types.Var)
	if !isVar {
		return false
	}

	// Parameters live in a function scope whose parent is the package scope.
	// Local variables also live in function scopes (or nested block scopes),
	// but we distinguish params by checking that the variable is declared in
	// a scope whose parent is the package scope (function body scope), and
	// the variable's position falls within the function signature, not the body.
	// A simpler heuristic: v.IsField() is false and v.Parent() is a function scope.
	// The types package does not expose "is parameter" directly, so we check
	// whether the Var appears in the function's signature Params.
	if !isParamVar(varObj, info) {
		return false
	}

	chanType, isChan := varObj.Type().(*types.Chan)
	if !isChan {
		return false
	}

	return chanType.Dir() == types.SendRecv
}

// isParamVar reports whether v is a function parameter by checking
// the enclosing function signature in the type information.
func isParamVar(varObj *types.Var, info *types.Info) bool {
	scope := varObj.Parent()
	if scope == nil {
		return false
	}

	// Walk all definitions looking for the function whose scope matches.
	for _, obj := range info.Defs {
		fn, isFn := obj.(*types.Func)
		if !isFn {
			continue
		}

		sig, isSig := fn.Type().(*types.Signature)
		if !isSig {
			continue
		}

		for param := range sig.Params().Variables() {
			if param == varObj {
				return true
			}
		}
	}

	return false
}
