package analyzers

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// errorsPkgPath is the import path for the standard errors package.
const errorsPkgPath = "errors"

// testFailMethods are the testing.TB methods that fail the current test.
// Their presence in an `if err == nil` body marks it as an "expected an
// error" assertion rather than a production happy-path branch.
var testFailMethods = map[string]bool{ //nolint:gochecknoglobals // package-internal lookup table
	"Fatal":   true,
	"Fatalf":  true,
	"FailNow": true,
	errorName: true,
	"Errorf":  true,
	"Fail":    true,
}

// redundantErrNilCheckAnalyzer returns an analyzer that flags a test-file
// `if err == nil { t.Fatal(...) }` assertion immediately followed by an
// `if` that runs errors.Is/As/AsType on the same err. The errors call already
// fails when err is nil, so the explicit nil check is redundant.
func redundantErrNilCheckAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "redundanterrnilcheck",
		Doc:      "flags an err == nil test assertion made redundant by a following errors.Is/As/AsType on the same err",
		Run:      runRedundantErrNilCheck,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runRedundantErrNilCheck(pass *analysis.Pass) (any, error) {
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

		if !isTestFile(pass, block) {
			return
		}

		checkBlockForRedundantNilCheck(pass, block)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkBlockForRedundantNilCheck scans consecutive statements for the pattern:
// an `if err == nil { <fails test> }` followed immediately by an `if` running
// errors.Is/As/AsType on the same err variable.
func checkBlockForRedundantNilCheck(pass *analysis.Pass, block *ast.BlockStmt) {
	for idx := range block.List {
		if idx+1 >= len(block.List) {
			break
		}

		nilCheck, isIf := block.List[idx].(*ast.IfStmt)
		if !isIf {
			continue
		}

		errObj, hasNilCheck := errEqNilObject(nilCheck.Cond, pass.TypesInfo)
		if !hasNilCheck {
			continue
		}

		// Only an assertion that fails when err is nil is redundant. A
		// production `if err == nil { return ok }` branch does real work.
		if !blockFailsTest(nilCheck.Body, pass.TypesInfo) {
			continue
		}

		next, isNextIf := block.List[idx+1].(*ast.IfStmt)
		if !isNextIf {
			continue
		}

		if !ifMatchesErrorsCall(next, pass.TypesInfo, errObj) {
			continue
		}

		pass.Reportf(
			nilCheck.Pos(),
			"redundant err == nil check; the following errors.Is/As/AsType on the same err already fails when err is nil",
		)
	}
}

// errEqNilObject reports whether cond is `x == nil` (in either operand order)
// and returns the object of the non-nil operand when it is a plain identifier.
func errEqNilObject(cond ast.Expr, info *types.Info) (types.Object, bool) {
	bin, isBinary := cond.(*ast.BinaryExpr)
	if !isBinary || bin.Op != token.EQL {
		return nil, false
	}

	if info.Types[bin.Y].IsNil() {
		return identObject(bin.X, info)
	}

	if info.Types[bin.X].IsNil() {
		return identObject(bin.Y, info)
	}

	return nil, false
}

// identObject returns the types object for expr when it is a plain identifier.
func identObject(expr ast.Expr, info *types.Info) (types.Object, bool) {
	ident, isIdent := expr.(*ast.Ident)
	if !isIdent {
		return nil, false
	}

	obj := info.ObjectOf(ident)
	if obj == nil {
		return nil, false
	}

	return obj, true
}

// blockFailsTest reports whether body contains a call to a testing.TB method
// that fails the test (Fatal, Error, FailNow, etc.).
func blockFailsTest(body *ast.BlockStmt, info *types.Info) bool {
	var found bool

	ast.Inspect(body, func(node ast.Node) bool {
		if found {
			return false
		}

		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}

		if isTestingFailureCall(call, info) {
			found = true

			return false
		}

		return true
	})

	return found
}

// isTestingFailureCall reports whether call is a failing method on a
// testing.T/B/common/TB receiver. The receiver-type check prevents matching
// unrelated methods such as the error interface's Error() or a logger's Error.
func isTestingFailureCall(call *ast.CallExpr, info *types.Info) bool {
	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel || !testFailMethods[sel.Sel.Name] {
		return false
	}

	return isTestingTBReceiver(sel.X, info)
}

// isTestingTBReceiver reports whether expr has a type from the testing package
// that carries the failure methods (T, B, common, or the TB interface).
func isTestingTBReceiver(expr ast.Expr, info *types.Info) bool {
	typ := info.TypeOf(expr)
	if typ == nil {
		return false
	}

	if ptr, isPtr := typ.(*types.Pointer); isPtr {
		typ = ptr.Elem()
	}

	named, isNamed := typ.(*types.Named)
	if !isNamed {
		return false
	}

	obj := named.Obj()
	if obj.Pkg() == nil || obj.Pkg().Path() != testingPkgPath {
		return false
	}

	switch obj.Name() {
	case "T", "B", "common", "TB":
		return true
	default:
		return false
	}
}

// ifMatchesErrorsCall reports whether ifStmt runs errors.Is/As/AsType on the
// same err object (errObj) in its init assignment or condition. For errors.Is
// the sentinel target must be non-nil, since errors.Is(err, nil) is itself the
// nil check.
func ifMatchesErrorsCall(ifStmt *ast.IfStmt, info *types.Info, errObj types.Object) bool {
	for _, expr := range ifStmtErrorsCandidates(ifStmt) {
		call, isCall := expr.(*ast.CallExpr)
		if !isCall {
			continue
		}

		if errorsCallMatches(call, info, errObj) {
			return true
		}
	}

	return false
}

// ifStmtErrorsCandidates returns the expressions in an if statement that may
// hold the errors call: the RHS of an init assignment and the condition
// (unwrapping a leading logical NOT).
func ifStmtErrorsCandidates(ifStmt *ast.IfStmt) []ast.Expr {
	var candidates []ast.Expr

	if assign, isAssign := ifStmt.Init.(*ast.AssignStmt); isAssign {
		candidates = append(candidates, assign.Rhs...)
	}

	cond := ifStmt.Cond
	if unary, isUnary := cond.(*ast.UnaryExpr); isUnary && unary.Op == token.NOT {
		cond = unary.X
	}

	candidates = append(candidates, cond)

	return candidates
}

// errorsCallMatches reports whether call is errors.Is/As/AsType with errObj as
// its first argument, excluding errors.Is(err, nil).
func errorsCallMatches(call *ast.CallExpr, info *types.Info, errObj types.Object) bool {
	name, isErrorsCall := errorsMatchFuncName(call, info)
	if !isErrorsCall {
		return false
	}

	if len(call.Args) == 0 {
		return false
	}

	firstObj, hasIdent := identObject(call.Args[0], info)
	if !hasIdent || firstObj != errObj {
		return false
	}

	// errors.Is(err, nil) is the nil check itself, not a redundant follow-up.
	if name == isFunc && (len(call.Args) < 2 || info.Types[call.Args[1]].IsNil()) {
		return false
	}

	return true
}

// errorsMatchFuncName returns the errors function name (Is, As, AsType) and
// whether call targets the standard errors package. It unwraps generic
// instantiation so errors.AsType[T](err) is recognized.
func errorsMatchFuncName(call *ast.CallExpr, info *types.Info) (string, bool) {
	fun := call.Fun

	switch indexed := fun.(type) {
	case *ast.IndexExpr:
		fun = indexed.X
	case *ast.IndexListExpr:
		fun = indexed.X
	}

	sel, isSel := fun.(*ast.SelectorExpr)
	if !isSel {
		return "", false
	}

	obj := info.ObjectOf(sel.Sel)
	if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != errorsPkgPath {
		return "", false
	}

	switch sel.Sel.Name {
	case "Is", "As", "AsType":
		return sel.Sel.Name, true
	default:
		return "", false
	}
}
