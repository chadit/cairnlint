package analyzers

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// httpClientTimeoutAnalyzer returns an analyzer that flags &http.Client{}
// composite literals missing a Timeout field. Without an explicit Timeout the
// client will wait indefinitely for slow or unresponsive servers, which can
// exhaust goroutines and file descriptors under load.
func httpClientTimeoutAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "httpclienttimeout",
		Doc:      "flags http.Client composite literals without a Timeout field",
		Run:      runHTTPClientTimeout,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runHTTPClientTimeout(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.CompositeLit)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		lit, isLit := node.(*ast.CompositeLit)
		if !isLit {
			return
		}

		// Skip test files where missing timeouts are acceptable.
		filename := pass.Fset.Position(lit.Pos()).Filename
		if strings.HasSuffix(filename, "_test.go") {
			return
		}

		if !isHTTPClientType(pass, lit) {
			return
		}

		if hasTimeoutField(lit) {
			return
		}

		pass.Reportf(lit.Pos(), "http.Client without Timeout; set an explicit Timeout to avoid hanging on slow servers")
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isHTTPClientType reports whether lit is a composite literal of type
// net/http.Client. Handles both http.Client{} and &http.Client{} (the
// UnaryExpr wrapper is transparent because the inspector visits the
// CompositeLit inside the &).
func isHTTPClientType(pass *analysis.Pass, lit *ast.CompositeLit) bool {
	typ := pass.TypesInfo.TypeOf(lit)
	if typ == nil {
		return false
	}

	// Unwrap pointer if present (e.g. &http.Client{} yields *http.Client).
	if ptr, isPtr := typ.(*types.Pointer); isPtr {
		typ = ptr.Elem()
	}

	named, isNamed := typ.(*types.Named)
	if !isNamed {
		return false
	}

	obj := named.Obj()

	return obj.Name() == "Client" && obj.Pkg() != nil && obj.Pkg().Path() == httpPkgPath
}

// hasTimeoutField reports whether lit contains a KeyValueExpr with key
// "Timeout".
func hasTimeoutField(lit *ast.CompositeLit) bool {
	for _, elt := range lit.Elts {
		kv, isKV := elt.(*ast.KeyValueExpr)
		if !isKV {
			continue
		}

		ident, isIdent := kv.Key.(*ast.Ident)
		if isIdent && ident.Name == "Timeout" {
			return true
		}
	}

	return false
}
