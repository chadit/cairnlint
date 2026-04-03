package analyzers

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// genericErrorMessages lists error message strings that are too vague to be
// useful for debugging or programmatic error handling.
var genericErrorMessages = []string{ //nolint:gochecknoglobals // lookup table, not mutable state
	"error",
	"failed",
	"operation failed",
	"something went wrong",
	"invalid",
	"invalid input",
	"an error occurred",
	"unknown error",
	"internal error",
}

// noGenericErrorAnalyzer returns an analyzer that flags errors.New() calls
// whose message is one of a set of known vague strings. Errors should carry
// enough context to identify what went wrong and where.
func noGenericErrorAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "nogenericerror",
		Doc:      "flags errors.New() with vague messages like \"error\" or \"failed\"; use descriptive error text",
		Run:      runNoGenericError,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runNoGenericError(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return
		}

		if !isCallTo(call, pass.TypesInfo, "errors", "New") {
			return
		}

		if len(call.Args) != 1 {
			return
		}

		lit, isLit := call.Args[0].(*ast.BasicLit)
		if !isLit || lit.Kind != token.STRING {
			return
		}

		msg := strings.Trim(lit.Value, "\"`")

		if isGenericErrorMessage(msg) {
			pass.Reportf(call.Pos(), "error message %s is too vague; provide context about what failed and why", lit.Value)
		}
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isGenericErrorMessage reports whether msg matches one of the known vague
// error strings (case-insensitive comparison).
func isGenericErrorMessage(msg string) bool {
	return slices.Contains(genericErrorMessages, strings.ToLower(msg))
}
