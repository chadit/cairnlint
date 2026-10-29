package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// writeMethodName is the io.Writer method that turns a []byte conversion into
// an Fprintf opportunity.
const writeMethodName = "Write"

// writeResultCount is the number of results on io.Writer.Write, used to tell
// that method apart from same-named methods with other shapes.
const writeResultCount = 2

// appendSpreadArgCount is the argument count of append(dst, src...), the only
// form Go allows with a spread.
const appendSpreadArgCount = 2

// preferFmtAppendfAnalyzer returns an analyzer that flags []byte(fmt.Sprintf(...))
// only where a direct replacement drops both the string and the slice: a Write
// argument becomes fmt.Fprintf, an append spread becomes fmt.Appendf(dst, ...).
// A bare conversion is left alone because upstream judged fmt.Appendf(nil, ...)
// less clear and it returns nil on empty output (golang/go#77581, #77557).
func preferFmtAppendfAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "preferfmtappendf",
		Doc:      "flags []byte(fmt.Sprintf(...)) passed to a Write call (use fmt.Fprintf) or spread into append (use fmt.Appendf(dst, ...))",
		Run:      runPreferFmtAppendf,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runPreferFmtAppendf(pass *analysis.Pass) (any, error) {
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
		if !isCall || !isSprintfByteConversion(call, pass.TypesInfo) {
			return true
		}

		if !goVersionAtLeast(pass, call.Pos(), goVersion119) {
			return true
		}

		if writeTarget := enclosingWriteCall(stack, pass.TypesInfo); writeTarget != "" {
			pass.Reportf(call.Pos(),
				"use fmt.Fprintf(%s, ...) instead of %s.Write([]byte(fmt.Sprintf(...))) to skip the intermediate string and byte slice",
				writeTarget, writeTarget)

			return true
		}

		if appendDst := enclosingAppendSpread(stack, pass.TypesInfo); appendDst != "" {
			pass.Reportf(call.Pos(),
				"use fmt.Appendf(%s, ...) instead of append(%s, []byte(fmt.Sprintf(...))...) to skip the intermediate string and byte slice",
				appendDst, appendDst)
		}

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isSprintfByteConversion reports whether call is []byte(fmt.Sprintf(...)).
func isSprintfByteConversion(call *ast.CallExpr, info *types.Info) bool {
	if !isByteSliceConversion(call) || len(call.Args) != 1 {
		return false
	}

	innerCall, isInnerCall := call.Args[0].(*ast.CallExpr)
	if !isInnerCall {
		return false
	}

	return isCallTo(innerCall, info, "fmt", "Sprintf")
}

// isByteSliceConversion reports whether call is a type conversion to []byte.
// The AST represents []byte(x) as a CallExpr where Fun is an ArrayType with
// nil Len and a byte Elt.
func isByteSliceConversion(call *ast.CallExpr) bool {
	arrayType, isArray := call.Fun.(*ast.ArrayType)
	if !isArray || arrayType.Len != nil {
		return false
	}

	eltIdent, isIdent := arrayType.Elt.(*ast.Ident)

	return isIdent && eltIdent.Name == "byte"
}

// enclosingWriteCall returns the source text of the receiver when the node on
// top of stack is the sole argument of a Write([]byte) (int, error) call, and
// the empty string otherwise.
func enclosingWriteCall(stack []ast.Node, info *types.Info) string {
	const parentDepth = 2
	if len(stack) < parentDepth {
		return ""
	}

	parent, isCall := stack[len(stack)-parentDepth].(*ast.CallExpr)
	if !isCall || len(parent.Args) != 1 || parent.Args[0] != stack[len(stack)-1] {
		return ""
	}

	sel, isSel := parent.Fun.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != writeMethodName {
		return ""
	}

	if !isWriteSignature(info.TypeOf(sel)) {
		return ""
	}

	recv, isIdent := sel.X.(*ast.Ident)
	if !isIdent {
		return ""
	}

	return recv.Name
}

// isWriteSignature reports whether typ is func([]byte) (int, error), the
// io.Writer method shape. Matching the signature rather than the name alone
// keeps unrelated Write methods out of the Fprintf suggestion.
func isWriteSignature(typ types.Type) bool {
	sig, isSig := typ.(*types.Signature)
	if !isSig || sig.Params().Len() != 1 || sig.Results().Len() != writeResultCount {
		return false
	}

	if !types.Identical(sig.Params().At(0).Type(), types.NewSlice(types.Typ[types.Byte])) {
		return false
	}

	if !types.Identical(sig.Results().At(0).Type(), types.Typ[types.Int]) {
		return false
	}

	return types.Identical(sig.Results().At(1).Type(), types.Universe.Lookup("error").Type())
}

// enclosingAppendSpread returns the source text of the destination when the
// node on top of stack is the final, spread argument of the builtin append,
// and the empty string otherwise. The destination needs no type check: a
// []byte spread only compiles when the destination is a byte slice.
func enclosingAppendSpread(stack []ast.Node, info *types.Info) string {
	const parentDepth = 2
	if len(stack) < parentDepth {
		return ""
	}

	parent, isCall := stack[len(stack)-parentDepth].(*ast.CallExpr)
	if !isCall || !parent.Ellipsis.IsValid() || len(parent.Args) != appendSpreadArgCount {
		return ""
	}

	if parent.Args[1] != stack[len(stack)-1] || !isBuiltinAppend(parent.Fun, info) {
		return ""
	}

	return types.ExprString(parent.Args[0])
}

// isBuiltinAppend reports whether fun names the predeclared append, so a
// user-defined function that shadows the name is not matched.
func isBuiltinAppend(fun ast.Expr, info *types.Info) bool {
	ident, isIdent := fun.(*ast.Ident)
	if !isIdent || ident.Name != "append" {
		return false
	}

	_, isBuiltin := info.Uses[ident].(*types.Builtin)

	return isBuiltin
}
