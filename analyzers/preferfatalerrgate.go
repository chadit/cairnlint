package analyzers

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// haltingFailMethods are testing.TB methods that stop the test goroutine via
// runtime.Goexit. Their presence means a missing error cannot fall through.
var haltingFailMethods = map[string]bool{ //nolint:gochecknoglobals // package-internal lookup table
	"Fatal":   true,
	"Fatalf":  true,
	"FailNow": true,
	"SkipNow": true,
	"Skip":    true,
	"Skipf":   true,
}

// nonHaltingFailMethods are testing.TB methods that record a failure but let
// execution continue past the check.
var nonHaltingFailMethods = map[string]bool{ //nolint:gochecknoglobals // package-internal lookup table
	errorName: true,
	"Errorf":  true,
	"Fail":    true,
}

// parentStackDepth is the number of trailing stack entries needed to reach a
// node's parent: the node itself plus its parent.
const parentStackDepth = 2

// preferFatalErrGateAnalyzer returns an analyzer that flags an
// errors.Is/As/AsType assertion whose failure branch uses a non-halting
// t.Error when the error (or the value extracted by As) is used after the
// check. Continuing past a wrong/absent error risks a nil dereference, so
// t.Fatal is the safer choice. Checks inside a spawned goroutine are skipped
// because t.Fatal must run on the test goroutine.
func preferFatalErrGateAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "preferfatalerrgate",
		Doc:      "flags non-halting t.Error in an errors.Is/As/AsType assertion when err is used afterward; prefer t.Fatal",
		Run:      runPreferFatalErrGate,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runPreferFatalErrGate(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.IfStmt)(nil),
	}

	insp.WithStack(nodeFilter, func(node ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		ifStmt, isIf := node.(*ast.IfStmt)
		if !isIf {
			return true
		}

		checkErrGate(pass, ifStmt, stack)

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkErrGate reports a single errors gate when its failure branch is
// non-halting and the guarded value is used after the check.
func checkErrGate(pass *analysis.Pass, ifStmt *ast.IfStmt, stack []ast.Node) {
	if !isTestFile(pass, ifStmt) {
		return
	}

	// t.Fatal must be called from the test goroutine, so a check inside a
	// `go func(){...}()` must not be rewritten to halt.
	if isInsideGoStmtClosure(stack) {
		return
	}

	block := parentBlock(stack)
	if block == nil {
		return
	}

	errObj, extracted, isGate := errorsGateInfo(ifStmt, pass.TypesInfo)
	if !isGate {
		return
	}

	if bodyHalts(ifStmt.Body, pass.TypesInfo) {
		return
	}

	failCall := firstNonHaltingFailCall(ifStmt.Body, pass.TypesInfo)
	if failCall == nil {
		return
	}

	guarded := []types.Object{errObj}
	if extracted != nil {
		guarded = append(guarded, extracted)
	}

	if !objectsUsedAfter(block, ifStmt, guarded, pass.TypesInfo) {
		return
	}

	pass.Reportf(
		failCall.Pos(),
		"errors assertion uses non-halting t.Error, but the error is used after this check; use t.Fatal to halt the test before a possible nil dereference",
	)
}

// errorsGateInfo reports whether ifStmt runs errors.Is/As/AsType on an err
// variable and returns that err object plus, for errors.As, the object the
// extracted error is stored into.
func errorsGateInfo(ifStmt *ast.IfStmt, info *types.Info) (types.Object, types.Object, bool) {
	for _, expr := range ifStmtErrorsCandidates(ifStmt) {
		call, isCall := expr.(*ast.CallExpr)
		if !isCall {
			continue
		}

		name, isErrorsCall := errorsMatchFuncName(call, info)
		if !isErrorsCall || len(call.Args) == 0 {
			continue
		}

		errObject, hasErr := identObject(call.Args[0], info)
		if !hasErr {
			continue
		}

		// errors.Is(err, nil) is just a nil check, not an error-identity gate.
		if name == isFunc && (len(call.Args) < parentStackDepth || info.Types[call.Args[1]].IsNil()) {
			continue
		}

		var target types.Object
		if name == "As" && len(call.Args) >= 2 {
			target, _ = addrOperandObject(call.Args[1], info)
		}

		return errObject, target, true
	}

	return nil, nil, false
}

// addrOperandObject returns the object behind an `&x` address-of expression.
func addrOperandObject(expr ast.Expr, info *types.Info) (types.Object, bool) {
	unary, isUnary := expr.(*ast.UnaryExpr)
	if !isUnary || unary.Op != token.AND {
		return nil, false
	}

	return identObject(unary.X, info)
}

// bodyHalts reports whether body stops fall-through, either via a Goexit-style
// testing call (Fatal, FailNow, Skip) or a trailing return/panic/branch.
func bodyHalts(body *ast.BlockStmt, info *types.Info) bool {
	halts := false

	ast.Inspect(body, func(node ast.Node) bool {
		if halts {
			return false
		}

		call, isCall := node.(*ast.CallExpr)
		if isCall && isTestingMethodCall(call, info, haltingFailMethods) {
			halts = true

			return false
		}

		return true
	})

	if halts {
		return true
	}

	return lastStmtTerminates(body)
}

// lastStmtTerminates reports whether the final statement of body returns,
// panics, or branches, which prevents code after the if from running.
func lastStmtTerminates(body *ast.BlockStmt) bool {
	if len(body.List) == 0 {
		return false
	}

	switch last := body.List[len(body.List)-1].(type) {
	case *ast.ReturnStmt, *ast.BranchStmt:
		return true
	case *ast.ExprStmt:
		call, isCall := last.X.(*ast.CallExpr)
		if !isCall {
			return false
		}

		ident, isIdent := call.Fun.(*ast.Ident)

		return isIdent && ident.Name == "panic"
	default:
		return false
	}
}

// firstNonHaltingFailCall returns the first non-halting failure call (t.Error,
// t.Errorf, t.Fail) inside body, or nil when there is none.
func firstNonHaltingFailCall(body *ast.BlockStmt, info *types.Info) *ast.CallExpr {
	var found *ast.CallExpr

	ast.Inspect(body, func(node ast.Node) bool {
		if found != nil {
			return false
		}

		call, isCall := node.(*ast.CallExpr)
		if isCall && isTestingMethodCall(call, info, nonHaltingFailMethods) {
			found = call

			return false
		}

		return true
	})

	return found
}

// objectsUsedAfter reports whether any of objs is referenced in a statement
// that follows ifStmt within the same block.
func objectsUsedAfter(block *ast.BlockStmt, ifStmt *ast.IfStmt, objs []types.Object, info *types.Info) bool {
	idx := blockIndexOf(block, ifStmt)
	if idx < 0 {
		return false
	}

	for _, stmt := range block.List[idx+1:] {
		if stmtUsesAnyObject(stmt, objs, info) {
			return true
		}
	}

	return false
}

// stmtUsesAnyObject reports whether node references any identifier resolving to
// one of objs.
func stmtUsesAnyObject(node ast.Node, objs []types.Object, info *types.Info) bool {
	var used bool

	ast.Inspect(node, func(inner ast.Node) bool {
		if used {
			return false
		}

		ident, isIdent := inner.(*ast.Ident)
		if !isIdent {
			return true
		}

		if obj := info.ObjectOf(ident); obj != nil && slices.Contains(objs, obj) {
			used = true

			return false
		}

		return true
	})

	return used
}

// blockIndexOf returns the index of target within block.List, or -1.
func blockIndexOf(block *ast.BlockStmt, target ast.Stmt) int {
	for idx, stmt := range block.List {
		if stmt == target {
			return idx
		}
	}

	return -1
}

// parentBlock returns the block statement directly enclosing the visited node,
// or nil when the parent is not a block (e.g., an else-if chain).
func parentBlock(stack []ast.Node) *ast.BlockStmt {
	if len(stack) < parentStackDepth {
		return nil
	}

	block, isBlock := stack[len(stack)-parentStackDepth].(*ast.BlockStmt)
	if !isBlock {
		return nil
	}

	return block
}

// isInsideGoStmtClosure reports whether the stack passes through a function
// literal that is the call target of a `go` statement.
func isInsideGoStmtClosure(stack []ast.Node) bool {
	for idx := range slices.Backward(stack) {
		funcLit, isFuncLit := stack[idx].(*ast.FuncLit)
		if !isFuncLit || idx < parentStackDepth {
			continue
		}

		call, isCall := stack[idx-1].(*ast.CallExpr)
		if !isCall || call.Fun != funcLit {
			continue
		}

		if goStmt, isGo := stack[idx-2].(*ast.GoStmt); isGo && goStmt.Call == call {
			return true
		}
	}

	return false
}

// isTestingMethodCall reports whether call is one of names called on a
// testing.T/B/common/TB receiver.
func isTestingMethodCall(call *ast.CallExpr, info *types.Info, names map[string]bool) bool {
	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel || !names[sel.Sel.Name] {
		return false
	}

	return isTestingTBReceiver(sel.X, info)
}
