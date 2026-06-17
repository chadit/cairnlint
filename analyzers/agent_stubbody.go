package analyzers

import (
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// notImplementedRe matches the canned messages an AI scaffold leaves behind when
// it writes a signature without an implementation. Kept broad on purpose: agent
// mode favors recall, and an LLM dismisses the rare false hit.
var notImplementedRe = regexp.MustCompile(`(?i)\b(not[ _-]?implemented|unimplemented|to[ _-]?do|stub|placeholder|fixme)\b`)

// errorType (the builtin error interface) is declared in typednilerror.go and
// reused here to spot canned error constructions by their result type.

// agentStubBodyAnalyzer flags functions whose body does nothing their name or
// doc claims: an empty body under a descriptive doc, a lone return of a canned
// value, or an explicit "not implemented" sentinel. These compile and pass the
// standard linters, so only a reader catches that the work is missing. It is
// agent-only because legitimate no-ops exist (interface satisfiers, null
// objects) and the heuristic cannot tell them from a stub.
func agentStubBodyAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "agentstubbody",
		Doc:  "[agent] flags functions whose body does nothing their name or doc claims (fake-done stubs)",
		Run:  runAgentStubBody,
	}
}

func runAgentStubBody(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		for _, decl := range file.Decls {
			funcDecl, isFunc := decl.(*ast.FuncDecl)
			if !isFunc {
				continue
			}

			checkStubBody(pass, funcDecl)
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkStubBody dispatches on the shape of the body. Only an empty body or a
// single bare return can be a stub; anything longer is doing work.
func checkStubBody(pass *analysis.Pass, funcDecl *ast.FuncDecl) {
	if funcDecl.Body == nil { // external or assembly declaration, no body to read
		return
	}

	if isTestFrameworkFunc(funcDecl.Name.Name) {
		return
	}

	hasDoc := funcDecl.Doc != nil && strings.TrimSpace(funcDecl.Doc.Text()) != ""

	switch {
	case len(funcDecl.Body.List) == 0:
		reportEmptyBody(pass, funcDecl, hasDoc)
	case len(funcDecl.Body.List) == 1:
		ret, isReturn := funcDecl.Body.List[0].(*ast.ReturnStmt)
		if isReturn {
			checkSingleReturn(pass, funcDecl, ret, hasDoc)
		}
	}
}

// reportEmptyBody flags an empty body only when a doc comment promises behavior.
// A void function with an empty body and no doc is an intentional no-op.
func reportEmptyBody(pass *analysis.Pass, funcDecl *ast.FuncDecl, hasDoc bool) {
	if !hasDoc {
		return
	}

	pass.Reportf(funcDecl.Name.Pos(),
		"[agent] %s %s has an empty body but its doc describes behavior: implement it or drop the claim",
		kindOf(funcDecl), funcDecl.Name.Name)
}

// checkSingleReturn flags a one-line return that hands back a canned value.
func checkSingleReturn(pass *analysis.Pass, funcDecl *ast.FuncDecl, ret *ast.ReturnStmt, hasDoc bool) {
	if text, isStub := notImplementedSentinel(pass, ret); isStub {
		pass.Reportf(funcDecl.Name.Pos(),
			"[agent] %s %s is a %q stub: wire up a real implementation before use",
			kindOf(funcDecl), funcDecl.Name.Name, text)

		return
	}

	if !allTrivialResults(pass, ret) {
		return
	}

	switch {
	case hasErrorResult(funcDecl) && (hasParams(funcDecl) || hasDoc):
		pass.Reportf(funcDecl.Name.Pos(),
			"[agent] %s %s returns without doing the work its signature implies: a one-line %s that can fail but never acts is usually an unfinished stub",
			kindOf(funcDecl), funcDecl.Name.Name, kindOf(funcDecl))
	case hasDoc && !hasErrorResult(funcDecl):
		pass.Reportf(funcDecl.Name.Pos(),
			"[agent] %s %s returns a canned value; verify it does what its doc claims rather than standing in as a stub",
			kindOf(funcDecl), funcDecl.Name.Name)
	}
}

// notImplementedSentinel returns the canned message and true when a returned
// error is constructed from a string that reads like an unfinished stub.
func notImplementedSentinel(pass *analysis.Pass, ret *ast.ReturnStmt) (string, bool) {
	for _, expr := range ret.Results {
		call, isCall := expr.(*ast.CallExpr)
		if !isCall {
			continue
		}

		text, isCanned := cannedErrorLiteral(pass, call)
		if isCanned && notImplementedRe.MatchString(text) {
			return text, true
		}
	}

	return "", false
}

// allTrivialResults reports whether every returned expression is a canned value.
// A naked return (no expressions) is trivial: it hands back zero-valued results.
func allTrivialResults(pass *analysis.Pass, ret *ast.ReturnStmt) bool {
	for _, expr := range ret.Results {
		if !isTrivialResult(pass, expr) {
			return false
		}
	}

	return true
}

// isTrivialResult reports whether expr is a canned value: nil, a literal, or an
// error built from a string literal. A computed expression (a delegating call,
// a selector, arithmetic) is real work and returns false.
func isTrivialResult(pass *analysis.Pass, expr ast.Expr) bool {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name == "nil"
	case *ast.BasicLit:
		return true
	case *ast.CallExpr:
		_, isCanned := cannedErrorLiteral(pass, typed)

		return isCanned
	}

	return false
}

// cannedErrorLiteral returns the message and true when call builds an error from
// a string literal (errors.New("..."), fmt.Errorf("...")). Matching on the error
// result type plus a literal first argument avoids hard-coding constructor names
// and skips delegating calls like return tx.Commit().
func cannedErrorLiteral(pass *analysis.Pass, call *ast.CallExpr) (string, bool) {
	resultType := pass.TypesInfo.TypeOf(call)
	if resultType == nil || !types.Identical(resultType, errorType) {
		return "", false
	}

	if len(call.Args) == 0 {
		return "", false
	}

	lit, isLit := call.Args[0].(*ast.BasicLit)
	if !isLit || lit.Kind != token.STRING {
		return "", false
	}

	return strings.Trim(lit.Value, "`\""), true
}

// hasErrorResult reports whether funcDecl declares a result of the builtin error
// type. A function that can fail but never acts is the clearest fake-done shape.
func hasErrorResult(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Type.Results == nil {
		return false
	}

	for _, field := range funcDecl.Type.Results.List {
		ident, isIdent := field.Type.(*ast.Ident)
		if isIdent && ident.Name == "error" {
			return true
		}
	}

	return false
}

// hasParams reports whether funcDecl takes at least one parameter. Inputs it
// never reads sharpen the case that a canned return is unfinished.
func hasParams(funcDecl *ast.FuncDecl) bool {
	return funcDecl.Type.Params != nil && len(funcDecl.Type.Params.List) > 0
}

// kindOf returns "method" for a function with a receiver, otherwise "func".
func kindOf(funcDecl *ast.FuncDecl) string {
	if funcDecl.Recv != nil {
		return "method"
	}

	return "func"
}
