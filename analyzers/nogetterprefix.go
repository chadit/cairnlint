package analyzers

import (
	"go/ast"
	"strings"
	"unicode"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// noGetterPrefixAnalyzer returns an analyzer that flags accessor functions and
// methods named with a Get or get prefix. Go convention names a getter after
// the value it returns (Counts, not GetCounts). To stay clear of lookups and
// fetches, only zero-argument functions that return a value are flagged.
func noGetterPrefixAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "nogetterprefix",
		Doc:      "flags a Get/get prefix on zero-arg accessors that return a value; name it Counts() not GetCounts()",
		Run:      runNoGetterPrefix,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runNoGetterPrefix(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		funcDecl, isFuncDecl := node.(*ast.FuncDecl)
		if !isFuncDecl {
			return
		}

		// A pure accessor takes no arguments and returns at least one value.
		if funcDecl.Type.Params != nil && len(funcDecl.Type.Params.List) > 0 {
			return
		}

		if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) == 0 {
			return
		}

		suggestion, hasPrefix := getterSuggestion(funcDecl.Name.Name)
		if !hasPrefix {
			return
		}

		pass.Reportf(funcDecl.Name.Pos(), "accessor %q has a Get prefix; name it %q instead", funcDecl.Name.Name, suggestion)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// getterSuggestion returns the name with a leading Get/get prefix removed,
// preserving the original export case. hasPrefix is false when name does not
// start with Get or get followed by an uppercase letter.
func getterSuggestion(name string) (string, bool) {
	for _, prefix := range []string{"Get", "get"} {
		rest, found := strings.CutPrefix(name, prefix)
		if !found || rest == "" {
			continue
		}

		first := rune(rest[0])
		if !unicode.IsUpper(first) {
			continue
		}

		if prefix == "get" {
			return string(unicode.ToLower(first)) + rest[1:], true
		}

		return rest, true
	}

	return "", false
}
