package analyzers

import (
	"go/ast"
	"go/token"
	"strings"
	"unicode"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// constMixedCapsAnalyzer returns an analyzer that flags constant names that
// borrow C-style conventions: an underscore (MAX_SIZE) or a k-prefix
// (kMaxSize). Go uses MixedCaps for every name, constants included.
func constMixedCapsAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "constmixedcaps",
		Doc:      "flags const names with an underscore or k-prefix; use MixedCaps (MaxSize not MAX_SIZE)",
		Run:      runConstMixedCaps,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runConstMixedCaps(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.GenDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		genDecl, isGen := node.(*ast.GenDecl)
		if !isGen || genDecl.Tok != token.CONST {
			return
		}

		for _, spec := range genDecl.Specs {
			valueSpec, isValue := spec.(*ast.ValueSpec)
			if !isValue {
				continue
			}

			for _, name := range valueSpec.Names {
				reportConstName(pass, name)
			}
		}
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// reportConstName flags a single constant name that uses an underscore or the
// C-style k-prefix.
func reportConstName(pass *analysis.Pass, name *ast.Ident) {
	if name.Name == "_" {
		return
	}

	if strings.Contains(name.Name, "_") {
		pass.Reportf(name.Pos(), "constant %q contains an underscore; use MixedCaps (e.g., MaxSize)", name.Name)

		return
	}

	if isKPrefixed(name.Name) {
		pass.Reportf(name.Pos(), "constant %q uses a k-prefix; use MixedCaps (e.g., MaxSize)", name.Name)
	}
}

// isKPrefixed reports whether name is the C-style kFoo form: a lowercase k
// directly followed by an uppercase letter.
func isKPrefixed(name string) bool {
	return len(name) >= 2 && name[0] == 'k' && unicode.IsUpper(rune(name[1]))
}
