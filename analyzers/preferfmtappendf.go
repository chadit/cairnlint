package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// writeMethodName is the io.Writer method that turns a []byte conversion into
// an Fprintf opportunity rather than an Appendf one.
const writeMethodName = "Write"

// writeResultCount is the number of results on io.Writer.Write, used to tell
// that method apart from same-named methods with other shapes.
const writeResultCount = 2

// preferFmtAppendfAnalyzer returns an analyzer that flags []byte(fmt.Sprintf(...))
// conversions. fmt.Appendf(nil, ...) writes directly into a byte slice and
// avoids the intermediate string allocation that the conversion requires.
//
// The replacement depends on where the conversion lands. Feeding the result
// straight into a Write call should become fmt.Fprintf, which skips the byte
// slice as well as the string. Suggesting Appendf everywhere was the objection
// that got the upstream fmtappendf modernizer pulled from `go fix` in Go 1.27
// (golang/go#77581), so the diagnostic names whichever replacement fits.
func preferFmtAppendfAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "preferfmtappendf",
		Doc:      "flags []byte(fmt.Sprintf(...)); use fmt.Appendf(nil, ...), or fmt.Fprintf when the result feeds a Write call",
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

		pass.Reportf(call.Pos(), "use fmt.Appendf(nil, ...) instead of []byte(fmt.Sprintf(...)) to avoid intermediate string allocation")

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
