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
	contextPkgPath = "context"
	httpPkgPath    = "net/http"
	stringsPkgPath = "strings"
	syncPkgPath    = "sync"
	reflectPkgPath = "reflect"
)

// Shared method and function names used across multiple analyzers.
const (
	benchmarkPrefix = "Benchmark"
	goMethodName    = "Go"
	fuzzPrefix      = "Fuzz"
	testPrefix      = "Test"
	containsFunc    = "Contains"
)

// callMatcher describes a function call to flag in test files.
type callMatcher struct {
	pkgPath  string
	funcName string
}

// synctestExemptConfig holds the parameters for building a scope-dependent
// analyzer that flags calls in test files with synctest closure exemption.
type synctestExemptConfig struct {
	name     string
	doc      string
	message  string
	matchers []callMatcher
}

// testCallWithSynctestExemption builds an analyzer that flags calls matching
// any of the provided matchers in test files, unless the call is inside a
// synctest.Test closure.
func testCallWithSynctestExemption(cfg synctestExemptConfig) *analysis.Analyzer {
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
			if !isCall {
				return true
			}

			if !isTestFile(pass, call) {
				return true
			}

			if !matchesAny(call, pass.TypesInfo, cfg.matchers) {
				return true
			}

			if isInsideSynctestClosure(stack, pass.TypesInfo) {
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

// isInsideSynctestClosure walks the stack looking for a function literal
// that is an argument to synctest.Test(). Returns true if found.
func isInsideSynctestClosure(stack []ast.Node, info *types.Info) bool {
	for idx := range slices.Backward(stack) {
		funcLit, isFuncLit := stack[idx].(*ast.FuncLit)
		if !isFuncLit {
			continue
		}

		if idx == 0 {
			return false
		}

		parentCall, isCall := stack[idx-1].(*ast.CallExpr)
		if !isCall {
			continue
		}

		if isCallTo(parentCall, info, "testing/synctest", "Test") && isFuncLitArg(parentCall, funcLit) {
			return true
		}
	}

	return false
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

	param := funcDecl.Type.Params.List[0]

	paramType := info.TypeOf(param.Type)
	if paramType == nil {
		return false
	}

	ptr, isPtr := paramType.(*types.Pointer)
	if !isPtr {
		return false
	}

	named, isNamed := ptr.Elem().(*types.Named)
	if !isNamed {
		return false
	}

	obj := named.Obj()

	return obj.Pkg() != nil && obj.Pkg().Path() == testingPkgPath && obj.Name() == "B"
}
