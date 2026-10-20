package analyzers

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// testingPkgPath is the import path for the standard testing package,
// extracted as a constant to satisfy goconst across benchmark analyzers.
const testingPkgPath = "testing"

// Shared package paths used across multiple analyzers.
const (
	bytesPkgPath    = "bytes"
	contextPkgPath  = "context"
	httpPkgPath     = "net/http"
	httptestPkgPath = "net/http/httptest"
	ioPkgPath       = "io"
	reflectPkgPath  = "reflect"
	stringsPkgPath  = "strings"
	synctestPkgPath = "testing/synctest"
	syncPkgPath     = "sync"
	timePkgPath     = "time"
	tlsPkgPath      = "crypto/tls"
	urlPkgPath      = "net/url"
)

// mainPkgName is the package name of an executable's entry package.
const mainPkgName = "main"

// Shared method and function names used across multiple analyzers.
const (
	benchmarkPrefix = "Benchmark"
	examplePrefix   = "Example"
	goMethodName    = "Go"
	fuzzPrefix      = "Fuzz"
	testPrefix      = "Test"
	containsFunc    = "Contains"
	errorName       = "Error"
	isFunc          = "Is"
)

// callMatcher describes a function call to flag in test files.
type callMatcher struct {
	pkgPath  string
	funcName string
}

// testCallConfig holds the parameters for building an analyzer that flags
// calls in test files. Every such analyzer exempts synctest.Test closures.
// The optional fields narrow the match further.
type testCallConfig struct {
	// argMatches, when set, must also accept the call's arguments. It leads
	// the struct so govet's fieldalignment sees the pointer words packed.
	argMatches func(*ast.CallExpr, *types.Info) bool

	name    string
	doc     string
	message string

	matchers []callMatcher

	// skipWhenArgOf lists outer calls that carry their own diagnostic for
	// this shape. A matched call that is the first argument of one of them
	// is left to that analyzer, so one line is not reported twice.
	skipWhenArgOf []callMatcher

	// suggestsTContext marks a message that names t.Context(). The call is
	// skipped below Go 1.24, where the method does not exist, in TestMain and
	// Example functions, which have no testing value, and in Cleanup callbacks,
	// where t.Context() is already canceled. Other test-file functions stay in
	// scope, since a helper can take a testing parameter.
	suggestsTContext bool
}

// testCallAnalyzer builds an analyzer that flags calls matching cfg in test
// files.
func testCallAnalyzer(cfg *testCallConfig) *analysis.Analyzer {
	run := func(pass *analysis.Pass) (any, error) {
		if !hasTestFiles(pass) {
			return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
		}

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
			if !isCall || !isTestFile(pass, call) || !cfg.matches(call, pass.TypesInfo) {
				return true
			}

			if cfg.exempt(pass, call, stack) {
				return true
			}

			pass.Reportf(call.Pos(), "%s", cfg.message)

			return true
		})

		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	return &analysis.Analyzer{
		Name:     cfg.name,
		Doc:      cfg.doc,
		Run:      run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

// matches reports whether call hits one of cfg.matchers and, when argMatches
// is set, passes it too.
func (cfg *testCallConfig) matches(call *ast.CallExpr, info *types.Info) bool {
	if !matchesAny(call, info, cfg.matchers) {
		return false
	}

	return cfg.argMatches == nil || cfg.argMatches(call, info)
}

// exempt reports whether a matched call sits where the message does not
// apply. Every analyzer built here skips synctest bubbles. A t.Context()
// suggestion is also skipped where the suggestsTContext doc says it cannot
// be followed.
func (cfg *testCallConfig) exempt(pass *analysis.Pass, call *ast.CallExpr, stack []ast.Node) bool {
	info := pass.TypesInfo

	if isInsideSynctestClosure(stack, info) || isFirstArgOfAny(stack, info, cfg.skipWhenArgOf) {
		return true
	}

	if !cfg.suggestsTContext {
		return false
	}

	if !goVersionAtLeast(pass, call.Pos(), goVersion124) {
		return true
	}

	return inTestMainOrExample(stack, info) || cleanupScopeFor(pass).contains(stack, info)
}

// isFirstArgOfAny reports whether the node on top of stack is the first
// argument of a call matching one of matchers.
func isFirstArgOfAny(stack []ast.Node, info *types.Info, matchers []callMatcher) bool {
	callIdx := len(stack) - 1
	if callIdx < 1 {
		return false
	}

	parent, isCall := stack[callIdx-1].(*ast.CallExpr)
	if !isCall || len(parent.Args) == 0 || parent.Args[0] != stack[callIdx] {
		return false
	}

	return matchesAny(parent, info, matchers)
}

// matchesAny reports whether call matches any of the provided matchers.
func matchesAny(call *ast.CallExpr, info *types.Info, matchers []callMatcher) bool {
	for _, mat := range matchers {
		if isCallTo(call, info, mat.pkgPath, mat.funcName) {
			return true
		}
	}

	return false
}

// isTestFile reports whether the position is inside a _test.go file.
func isTestFile(pass *analysis.Pass, pos ast.Node) bool {
	return strings.HasSuffix(pass.Fset.Position(pos.Pos()).Filename, "_test.go")
}

// isMockPath reports whether filename lives in a directory named `mocks`.
// Matches any layout like `test/mocks/`, `tests/mocks/`, `internal/mocks/`,
// or a repo-root `mocks/` directory. Used to exempt mock infrastructure from
// rules that don't apply to test doubles.
func isMockPath(filename string) bool {
	dir := filepath.Dir(filename)

	return slices.Contains(strings.Split(dir, string(filepath.Separator)), "mocks")
}

// hasTestFiles reports whether the pass contains any _test.go files.
func hasTestFiles(pass *analysis.Pass) bool {
	for _, file := range pass.Files {
		if strings.HasSuffix(pass.Fset.Position(file.Pos()).Filename, "_test.go") {
			return true
		}
	}

	return false
}

// isCallTo reports whether call is a call to pkgPath.funcName.
// Uses type info to resolve through aliases and dot-imports.
func isCallTo(call *ast.CallExpr, info *types.Info, pkgPath, funcName string) bool {
	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != funcName {
		return false
	}

	obj := info.ObjectOf(sel.Sel)
	if obj == nil {
		return false
	}

	pkg := obj.Pkg()

	return pkg != nil && pkg.Path() == pkgPath
}

// isInsideSynctestClosure reports whether the node on top of stack sits in a
// function literal passed to synctest.Test.
func isInsideSynctestClosure(stack []ast.Node, info *types.Info) bool {
	return isInsideClosureArgOf(stack, func(call *ast.CallExpr) bool {
		return isCallTo(call, info, synctestPkgPath, "Test")
	})
}

// isInsideClosureArgOf walks the stack looking for a function literal that is
// an argument to a call accepted by isTarget.
func isInsideClosureArgOf(stack []ast.Node, isTarget func(*ast.CallExpr) bool) bool {
	for idx := range slices.Backward(stack) {
		funcLit, isFuncLit := stack[idx].(*ast.FuncLit)
		if !isFuncLit || idx == 0 {
			continue
		}

		parentCall, isCall := stack[idx-1].(*ast.CallExpr)
		if !isCall {
			continue
		}

		if isTarget(parentCall) && isFuncLitArg(parentCall, funcLit) {
			return true
		}
	}

	return false
}

// isTestingType reports whether typ is testing.T, testing.B, testing.F, or
// testing.TB. These are the types that carry Context(), Cleanup(), and the
// rest of the per-test API.
func isTestingType(typ types.Type) bool {
	switch testingTypeName(typ) {
	case "T", "B", "F", "TB":
		return true
	default:
		return false
	}
}

// testingTypeName returns the name of the testing package type behind typ,
// looking through one pointer and any aliases, or "" for any other type.
// Callers pick the names they accept, so one lookup serves the receiver
// checks, the parameter checks, and the benchmark signature check.
func testingTypeName(typ types.Type) string {
	if typ == nil {
		return ""
	}

	typ = types.Unalias(typ)
	if ptr, isPtr := typ.(*types.Pointer); isPtr {
		typ = types.Unalias(ptr.Elem())
	}

	named, isNamed := typ.(*types.Named)
	if !isNamed || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != testingPkgPath {
		return ""
	}

	return named.Obj().Name()
}

// isTestingBExpr reports whether expr has type *testing.B.
func isTestingBExpr(expr ast.Expr, info *types.Info) bool {
	return testingTypeName(info.TypeOf(expr)) == "B"
}

// inTestMainOrExample reports whether the node on top of stack sits in
// TestMain or an Example function. Neither receives a testing.T, B, or F, so
// t.Context() advice cannot be followed there.
func inTestMainOrExample(stack []ast.Node, info *types.Info) bool {
	for _, node := range stack {
		funcDecl, isDecl := node.(*ast.FuncDecl)
		if !isDecl {
			continue
		}

		return strings.HasPrefix(funcDecl.Name.Name, examplePrefix) || hasTestingMParam(funcDecl.Type, info)
	}

	return false
}

// hasTestingMParam reports whether funcType takes a *testing.M, the TestMain
// signature.
func hasTestingMParam(funcType *ast.FuncType, info *types.Info) bool {
	if funcType.Params == nil {
		return false
	}

	for _, field := range funcType.Params.List {
		if testingTypeName(info.TypeOf(field.Type)) == "M" {
			return true
		}
	}

	return false
}

// exprStmtCall returns the call expression when stmt is a bare call whose
// results are discarded, and nil for any other statement.
func exprStmtCall(stmt ast.Stmt) *ast.CallExpr {
	exprStmt, isExpr := stmt.(*ast.ExprStmt)
	if !isExpr {
		return nil
	}

	call, isCall := exprStmt.X.(*ast.CallExpr)
	if !isCall {
		return nil
	}

	return call
}

// isFuncLitArg reports whether lit appears as an argument in call.
func isFuncLitArg(call *ast.CallExpr, lit *ast.FuncLit) bool {
	for _, arg := range call.Args {
		if arg == lit {
			return true
		}
	}

	return false
}

// testingParam reports whether field is a parameter of pointer type to
// testing.T, testing.B, or testing.F, returning the parameter's variable name
// when it has exactly one. The name is empty for unnamed parameters.
func testingParam(info *types.Info, field *ast.Field) (string, bool) {
	switch testingTypeName(info.TypeOf(field.Type)) {
	case "T", "B", "F":
	default:
		return "", false
	}

	var name string
	if len(field.Names) == 1 {
		name = field.Names[0].Name
	}

	return name, true
}

// receiverInfo extracts the type name, receiver variable name, and pointer-ness
// from a method receiver field. The final result is false when the receiver
// type is not a plain (possibly generic) named type.
func receiverInfo(field *ast.Field) (string, string, bool, bool) {
	expr := field.Type

	var isPtr bool
	if star, isStar := expr.(*ast.StarExpr); isStar {
		isPtr = true
		expr = star.X
	}

	var typeName string

	switch base := expr.(type) {
	case *ast.Ident:
		typeName = base.Name
	case *ast.IndexExpr:
		if id, isID := base.X.(*ast.Ident); isID {
			typeName = id.Name
		}
	case *ast.IndexListExpr:
		if id, isID := base.X.(*ast.Ident); isID {
			typeName = id.Name
		}
	}

	if typeName == "" {
		return "", "", false, false
	}

	var recvName string
	if len(field.Names) > 0 {
		recvName = field.Names[0].Name
	}

	return typeName, recvName, isPtr, true
}

// isBenchmarkFunc reports whether funcDecl is a benchmark function: name starts
// with "Benchmark" and has a single parameter of type *testing.B.
func isBenchmarkFunc(funcDecl *ast.FuncDecl, info *types.Info) bool {
	if funcDecl.Name == nil || len(funcDecl.Name.Name) <= len(benchmarkPrefix) {
		return false
	}

	if funcDecl.Name.Name[:len(benchmarkPrefix)] != benchmarkPrefix {
		return false
	}

	if funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) != 1 {
		return false
	}

	return isTestingBExpr(funcDecl.Type.Params.List[0].Type, info)
}
