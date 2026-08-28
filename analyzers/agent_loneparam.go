package analyzers

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// nilArgument is the key a nil argument contributes to the shared-value
// comparison, since an untyped nil carries no constant.Value to compare.
const nilArgument = "nil"

type calleeUsage struct {
	decl    *ast.FuncDecl
	callSet [][]ast.Expr
	escapes bool
}

// agentLoneParamAnalyzer flags a parameter that receives the same constant at
// every call site in the package. A parameter with one value is a knob nothing
// turns: the branch it selects never varies, so the parameter, the argument at
// each call site, and the unreachable half of the body all come out together.
//
// It is agent-only because the count is package-scoped and the guards are
// heuristic. Only unexported callees are considered, since an exported one can
// be called from anywhere, and a callee whose name is ever used outside a call
// is skipped because its arguments are then invisible. An LLM confirms the
// value never varies before acting.
func agentLoneParamAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "agentloneparam",
		Doc:  "[agent] flags parameters that receive the same constant at every call site in the package",
		Run:  runAgentLoneParam,
	}
}

func runAgentLoneParam(pass *analysis.Pass) (any, error) {
	if skipUncountablePass(pass) {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	usage := collectCallees(pass)
	recordReferences(pass, usage)

	for callee, seen := range usage {
		reportLoneParams(pass, callee, seen)
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// Declarations come from production files only, so a helper that exists to
// serve a test is not reported, while call sites below still count test files.
func collectCallees(pass *analysis.Pass) map[*types.Func]*calleeUsage {
	usage := make(map[*types.Func]*calleeUsage)

	for _, file := range pass.Files {
		if isTestFile(pass, file) {
			continue
		}

		for _, decl := range file.Decls {
			addCallee(pass, decl, usage)
		}
	}

	return usage
}

func addCallee(pass *analysis.Pass, decl ast.Decl, usage map[*types.Func]*calleeUsage) {
	funcDecl, isFunc := decl.(*ast.FuncDecl)
	if !isFunc || funcDecl.Body == nil || funcDecl.Name.IsExported() {
		return
	}

	if funcDecl.Type.Params.NumFields() == 0 || funcDecl.Type.TypeParams != nil || isVariadic(funcDecl) {
		return
	}

	obj, defined := pass.TypesInfo.Defs[funcDecl.Name].(*types.Func)
	if defined {
		usage[obj] = &calleeUsage{decl: funcDecl}
	}
}

// A variadic tail has no fixed argument position to compare across call sites.
func isVariadic(funcDecl *ast.FuncDecl) bool {
	params := funcDecl.Type.Params.List
	if len(params) == 0 {
		return false
	}

	_, isEllipsis := params[len(params)-1].Type.(*ast.Ellipsis)

	return isEllipsis
}

// Arguments are only visible where the call is written out, so a callee whose
// name is passed as a value could be invoked with anything and is dropped.
func recordReferences(pass *analysis.Pass, usage map[*types.Func]*calleeUsage) {
	callees := make(map[*ast.Ident]bool)

	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			call, isCall := node.(*ast.CallExpr)
			if isCall {
				markCall(pass, call, callees, usage)
			}

			return true
		})
	}

	markEscapes(pass, callees, usage)
}

func markCall(pass *analysis.Pass, call *ast.CallExpr, callees map[*ast.Ident]bool, usage map[*types.Func]*calleeUsage) {
	ident := identOf(call.Fun)
	if ident == nil {
		return
	}

	callees[ident] = true

	obj, isFunc := pass.TypesInfo.Uses[ident].(*types.Func)
	if !isFunc {
		return
	}

	seen, tracked := usage[obj]
	if tracked {
		seen.callSet = append(seen.callSet, call.Args)
	}
}

func markEscapes(pass *analysis.Pass, callees map[*ast.Ident]bool, usage map[*types.Func]*calleeUsage) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			ident, isIdent := node.(*ast.Ident)
			if isIdent && !callees[ident] {
				markEscape(pass, ident, usage)
			}

			return true
		})
	}
}

func markEscape(pass *analysis.Pass, ident *ast.Ident, usage map[*types.Func]*calleeUsage) {
	obj, isFunc := pass.TypesInfo.Uses[ident].(*types.Func)
	if !isFunc {
		return
	}

	seen, tracked := usage[obj]
	if tracked {
		seen.escapes = true
	}
}

func identOf(expr ast.Expr) *ast.Ident {
	switch node := expr.(type) {
	case *ast.Ident:
		return node
	case *ast.SelectorExpr:
		return node.Sel
	}

	return nil
}

// A single call site proves nothing about variation, so two are the minimum
// before "always receives" means anything.
func reportLoneParams(pass *analysis.Pass, callee *types.Func, seen *calleeUsage) {
	if seen.escapes || len(seen.callSet) < 2 {
		return
	}

	for index, name := range paramPositions(seen.decl) {
		shared, fixed := sharedConstant(pass, seen.callSet, index)
		if !fixed {
			continue
		}

		pass.Reportf(name.Pos(), "parameter %s of %s receives %s at all %d call sites in this package; the choice it encodes is never made", name.Name, callee.Name(), shared, len(seen.callSet))
	}
}

// A blank parameter the body cannot read is already another linter's finding,
// so it is left out rather than reported twice.
func paramPositions(funcDecl *ast.FuncDecl) map[int]*ast.Ident {
	positions := make(map[int]*ast.Ident)
	index := 0

	for _, field := range funcDecl.Type.Params.List {
		for _, name := range field.Names {
			if name.Name != "_" {
				positions[index] = name
			}

			index++
		}
	}

	return positions
}

func sharedConstant(pass *analysis.Pass, callSet [][]ast.Expr, index int) (string, bool) {
	var shared string

	for _, args := range callSet {
		if index >= len(args) {
			return "", false
		}

		value, isConst := constantKey(pass, args[index])
		if !isConst || (shared != "" && value != shared) {
			return "", false
		}

		shared = value
	}

	return shared, shared != ""
}

func constantKey(pass *analysis.Pass, arg ast.Expr) (string, bool) {
	info, known := pass.TypesInfo.Types[arg]
	if !known {
		return "", false
	}

	if info.Value != nil {
		return trimConstant(info.Value.ExactString()), true
	}

	if info.IsNil() {
		return nilArgument, true
	}

	return "", false
}

// A rational constant prints as a fraction, which reads badly in a diagnostic.
func trimConstant(exact string) string {
	if strings.Contains(exact, "/") {
		return strings.SplitN(exact, "/", 2)[0]
	}

	return exact
}
