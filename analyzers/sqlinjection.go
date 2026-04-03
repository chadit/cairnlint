package analyzers

import (
	"go/ast"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// sqlKeywordPattern matches SQL keywords in a string literal, used to detect
// format strings that build SQL queries via fmt.Sprintf.
var sqlKeywordPattern = regexp.MustCompile(`(?i)\b(SELECT|INSERT|UPDATE|DELETE|DROP|ALTER)\b`)

// sqlInjectionAnalyzer returns an analyzer that flags fmt.Sprintf calls whose
// format string contains SQL keywords. Building SQL with string formatting is
// an injection risk; use parameterized queries instead.
func sqlInjectionAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "sqlinjection",
		Doc:  "flags fmt.Sprintf with SQL keywords in the format string; use parameterized queries",
		Run:  runSQLInjection,
	}
}

func runSQLInjection(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			call, isCall := node.(*ast.CallExpr)
			if !isCall {
				return true
			}

			if !isFmtSprintf(call) {
				return true
			}

			if len(call.Args) == 0 {
				return true
			}

			formatStr := extractStringLiteral(call.Args[0])
			if formatStr == "" {
				return true
			}

			if sqlKeywordPattern.MatchString(formatStr) {
				pass.Reportf(call.Pos(), "SQL string formatting detected; use parameterized queries instead of fmt.Sprintf")
			}

			return true
		})
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isFmtSprintf reports whether call is a call to fmt.Sprintf.
func isFmtSprintf(call *ast.CallExpr) bool {
	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel {
		return false
	}

	ident, isIdent := sel.X.(*ast.Ident)
	if !isIdent {
		return false
	}

	return ident.Name == "fmt" && sel.Sel.Name == "Sprintf"
}

// extractStringLiteral returns the unquoted value of a basic string literal,
// or empty string if the expression is not a string literal.
func extractStringLiteral(expr ast.Expr) string {
	lit, isLit := expr.(*ast.BasicLit)
	if !isLit {
		return ""
	}

	val := lit.Value
	if len(val) >= 2 && strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
		return val[1 : len(val)-1]
	}

	return ""
}
