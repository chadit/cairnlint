package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// bodyField is the http.Response field holding the response stream.
const bodyField = "Body"

// redundantBodyDrainAnalyzer returns an analyzer that flags a manual drain of
// an HTTP response body whose result is thrown away.
//
// From Go 1.27 an HTTP/1 Response.Body drains its own unread content on Close,
// so the copy exists only to keep the connection reusable and no longer earns
// its place. The automatic drain stops at a conservative limit, so a caller
// that deliberately drains bodies larger than that limit still needs the copy;
// keeping the result of the copy, rather than discarding it, is enough to take
// the call out of scope here.
func redundantBodyDrainAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "redundantbodydrain",
		Doc:      "flags io.Copy(io.Discard, resp.Body) drains that Response.Body performs itself on Close (Go 1.27+)",
		Run:      runRedundantBodyDrain,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runRedundantBodyDrain(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.ExprStmt)(nil),
		(*ast.AssignStmt)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		call := discardedResultCall(node)
		if call == nil || !isResponseBodyDrain(call, pass.TypesInfo) {
			return
		}

		if !goVersionAtLeast(pass, call.Pos(), goVersion127) {
			return
		}

		pass.Reportf(call.Pos(),
			"HTTP/1 Response.Body drains unread content on Close from Go 1.27; drop this copy unless you rely on draining past the automatic limit")
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// discardedResultCall returns the call in stmt when every result it produces is
// thrown away, either by a bare call statement or by assigning to blanks.
//
// A caller that keeps the byte count or the error is doing something with the
// drain beyond connection reuse, so those forms are left alone.
func discardedResultCall(node ast.Node) *ast.CallExpr {
	switch stmt := node.(type) {
	case *ast.ExprStmt:
		call, isCall := stmt.X.(*ast.CallExpr)
		if !isCall {
			return nil
		}

		return call
	case *ast.AssignStmt:
		return blankAssignCall(stmt)
	default:
		return nil
	}
}

// blankAssignCall returns the right-hand call when an assignment discards every
// result into a blank identifier.
func blankAssignCall(assign *ast.AssignStmt) *ast.CallExpr {
	if len(assign.Rhs) != 1 {
		return nil
	}

	for _, lhs := range assign.Lhs {
		ident, isIdent := lhs.(*ast.Ident)
		if !isIdent || ident.Name != "_" {
			return nil
		}
	}

	call, isCall := assign.Rhs[0].(*ast.CallExpr)
	if !isCall {
		return nil
	}

	return call
}

// isResponseBodyDrain reports whether call copies an http.Response body into
// io.Discard.
func isResponseBodyDrain(call *ast.CallExpr, info *types.Info) bool {
	if !isCallTo(call, info, ioPkgPath, "Copy") || len(call.Args) != 2 {
		return false
	}

	return isIODiscard(call.Args[0], info) && isResponseBodyField(call.Args[1], info)
}

// isIODiscard reports whether expr denotes the io.Discard writer.
func isIODiscard(expr ast.Expr, info *types.Info) bool {
	sel, isSel := expr.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != "Discard" {
		return false
	}

	obj := info.ObjectOf(sel.Sel)

	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == ioPkgPath
}

// isResponseBodyField reports whether expr selects Body from an http.Response.
func isResponseBodyField(expr ast.Expr, info *types.Info) bool {
	sel, isSel := expr.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != bodyField {
		return false
	}

	recv := info.TypeOf(sel.X)
	if ptr, isPtr := recv.(*types.Pointer); isPtr {
		recv = ptr.Elem()
	}

	return isNamedType(recv, httpPkgPath, "Response")
}
