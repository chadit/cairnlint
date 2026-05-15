package analyzers

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// wgAddBeforeGoAnalyzer returns an analyzer that flags wg.Add(N) calls
// before wg.Go(...) on the same WaitGroup. Go 1.25's WaitGroup.Go already
// calls Add(1) internally, so a preceding Add double-counts the WaitGroup
// and causes Wait to hang forever.
func wgAddBeforeGoAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "wgaddbeforego",
		Doc:      "flags wg.Add() before wg.Go(); WaitGroup.Go already calls Add(1) internally, causing a double-count that hangs Wait",
		Run:      runWGAddBeforeGo,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runWGAddBeforeGo(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.BlockStmt)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		block, isBlock := node.(*ast.BlockStmt)
		if !isBlock {
			return
		}

		checkBlockForAddBeforeGo(block.List, pass)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkBlockForAddBeforeGo scans statements looking for wg.Add(N) followed
// by wg.Go(...) where both target the same variable. Looks ahead past
// harmless statements but stops at control flow that could consume the Add.
func checkBlockForAddBeforeGo(stmts []ast.Stmt, pass *analysis.Pass) {
	for idx, stmt := range stmts {
		addCall, addReceiver := extractWGAddCall(stmt, pass.TypesInfo)
		if addReceiver == "" {
			continue
		}

		// Look ahead in the same block for a wg.Go call on the same receiver.
		// Stop if we see something that might consume the Add, like another
		// goroutine-starting statement, a Wait call, or another Add.
		for ahead := idx + 1; ahead < len(stmts); ahead++ {
			nextStmt := stmts[ahead]

			// The statement itself or something nested inside it calls wg.Go.
			if containsWGGoCall(nextStmt, addReceiver, pass.TypesInfo) {
				pass.Reportf(addCall.Pos(), "wg.Add before wg.Go is redundant; WaitGroup.Go calls Add(1) internally, this double-counts and hangs Wait")

				break
			}

			// Another wg.Add on the same receiver shadows this one.
			_, anotherAddRecv := extractWGAddCall(nextStmt, pass.TypesInfo)
			if anotherAddRecv == addReceiver {
				break
			}

			// wg.Wait or wg.Done means the preceding Add was for something else.
			if stmtConsumesWG(nextStmt, addReceiver, pass.TypesInfo) {
				break
			}

			// A go or defer statement is a candidate for consuming the manual Add.
			if stmtStartsGoroutineOrDefer(nextStmt) {
				break
			}
		}
	}
}

// containsWGGoCall reports whether stmt contains any wg.Go call on the
// specified receiver. Recurses into nested blocks (loops, if-bodies) but
// does not enter function literals since they run in a different scope.
func containsWGGoCall(stmt ast.Stmt, receiver string, info *types.Info) bool {
	var found bool

	ast.Inspect(stmt, func(node ast.Node) bool {
		if node == nil || found {
			return false
		}

		if _, isFunc := node.(*ast.FuncLit); isFunc {
			return false
		}

		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}

		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel || sel.Sel.Name != goMethodName {
			return true
		}

		if !isWaitGroupReceiver(sel.X, info) && !isWaitGroupMethod(sel, info) {
			return true
		}

		if receiverIdent(sel.X) == receiver {
			found = true

			return false
		}

		return true
	})

	return found
}

// stmtConsumesWG reports whether stmt calls Wait or Done on the receiver.
func stmtConsumesWG(stmt ast.Stmt, receiver string, info *types.Info) bool {
	var found bool

	ast.Inspect(stmt, func(node ast.Node) bool {
		if node == nil || found {
			return false
		}

		if _, isFunc := node.(*ast.FuncLit); isFunc {
			return false
		}

		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}

		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel {
			return true
		}

		if sel.Sel.Name != "Wait" && sel.Sel.Name != "Done" {
			return true
		}

		if !isWaitGroupReceiver(sel.X, info) && !isWaitGroupMethod(sel, info) {
			return true
		}

		if receiverIdent(sel.X) == receiver {
			found = true

			return false
		}

		return true
	})

	return found
}

// stmtStartsGoroutineOrDefer reports whether stmt contains a go or defer
// statement (outside function literals).
func stmtStartsGoroutineOrDefer(stmt ast.Stmt) bool {
	var found bool

	ast.Inspect(stmt, func(node ast.Node) bool {
		if node == nil || found {
			return false
		}

		if _, isFunc := node.(*ast.FuncLit); isFunc {
			return false
		}

		switch node.(type) {
		case *ast.GoStmt, *ast.DeferStmt:
			found = true

			return false
		}

		return true
	})

	return found
}

// extractWGAddCall checks whether stmt is an expression statement calling
// Add on a *sync.WaitGroup receiver. Returns the call expression and the
// receiver identifier name, or (nil, "") if no match. Also detects promoted
// Add methods from embedded WaitGroups.
func extractWGAddCall(stmt ast.Stmt, info *types.Info) (*ast.CallExpr, string) {
	exprStmt, isExpr := stmt.(*ast.ExprStmt)
	if !isExpr {
		return nil, ""
	}

	call, isCall := exprStmt.X.(*ast.CallExpr)
	if !isCall {
		return nil, ""
	}

	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != "Add" {
		return nil, ""
	}

	if !isWaitGroupReceiver(sel.X, info) && !isWaitGroupMethod(sel, info) {
		return nil, ""
	}

	receiverName := receiverIdent(sel.X)
	if receiverName == "" {
		return nil, ""
	}

	return call, receiverName
}

// isWaitGroupReceiver reports whether expr resolves to a sync.WaitGroup
// type (either *sync.WaitGroup or sync.WaitGroup).
func isWaitGroupReceiver(expr ast.Expr, info *types.Info) bool {
	recvType := info.TypeOf(expr)
	if recvType == nil {
		return false
	}

	// Unwrap pointer if present
	if ptr, isPtr := recvType.(*types.Pointer); isPtr {
		recvType = ptr.Elem()
	}

	named, isNamed := recvType.(*types.Named)
	if !isNamed {
		return false
	}

	obj := named.Obj()

	return obj.Pkg() != nil && obj.Pkg().Path() == syncPkgPath && obj.Name() == "WaitGroup"
}

// isWaitGroupMethod reports whether the method selected by sel was declared
// on sync.WaitGroup. This catches promoted methods from embedded WaitGroups
// where the receiver type itself is not sync.WaitGroup.
func isWaitGroupMethod(sel *ast.SelectorExpr, info *types.Info) bool {
	obj := info.ObjectOf(sel.Sel)
	if obj == nil {
		return false
	}

	fn, isFn := obj.(*types.Func)
	if !isFn {
		return false
	}

	sig, isSig := fn.Type().(*types.Signature)
	if !isSig || sig.Recv() == nil {
		return false
	}

	recvType := sig.Recv().Type()
	if ptr, isPtr := recvType.(*types.Pointer); isPtr {
		recvType = ptr.Elem()
	}

	named, isNamed := recvType.(*types.Named)
	if !isNamed {
		return false
	}

	declObj := named.Obj()

	return declObj.Pkg() != nil && declObj.Pkg().Path() == syncPkgPath && declObj.Name() == "WaitGroup"
}

// receiverIdent extracts a string representation of the receiver expression
// for identity comparison. Two calls with the same receiverIdent string are
// assumed to target the same WaitGroup. This is heuristic, not proof, but
// matches what humans write in practice.
func receiverIdent(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.SelectorExpr:
		xName := receiverIdent(node.X)
		if xName != "" {
			return xName + "." + node.Sel.Name
		}
	case *ast.ParenExpr:
		return receiverIdent(node.X)
	case *ast.StarExpr:
		return "*" + receiverIdent(node.X)
	case *ast.UnaryExpr:
		// Handle &wg if it appears (unusual for method calls)
		return node.Op.String() + receiverIdent(node.X)
	case *ast.IndexExpr:
		// Handle map/slice access like m["key"] or wgs[0]
		collection := receiverIdent(node.X)

		index := exprString(node.Index)
		if collection != "" && index != "" {
			return collection + "[" + index + "]"
		}
	case *ast.CallExpr:
		// Handle function-return receivers like getWG()
		fnName := receiverIdent(node.Fun)
		if fnName != "" {
			return fnName + "(" + argsString(node.Args) + ")"
		}
	}

	return ""
}

// exprString produces a stable string for simple expressions used as map
// keys or slice indices. Returns "" for anything too complex to reliably
// compare by string.
func exprString(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.BasicLit:
		return node.Value
	case *ast.Ident:
		return node.Name
	case *ast.SelectorExpr:
		xName := exprString(node.X)
		if xName != "" {
			return xName + "." + node.Sel.Name
		}
	}

	return ""
}

// argsString produces a stable string for a function's argument list.
// Only handles simple expressions; returns "" if any argument is too complex.
func argsString(args []ast.Expr) string {
	if len(args) == 0 {
		return ""
	}

	const avgArgLen = 8 // rough estimate: short identifier per arg plus separator

	var buf strings.Builder
	buf.Grow(len(args) * avgArgLen)

	for idx, arg := range args {
		argStr := exprString(arg)
		if argStr == "" {
			return ""
		}

		if idx > 0 {
			buf.WriteString(", ")
		}

		buf.WriteString(argStr)
	}

	return buf.String()
}
