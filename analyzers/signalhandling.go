package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// signalHandlingAnalyzer returns an analyzer that flags main() functions
// starting HTTP servers or network listeners without signal handling.
// Without graceful shutdown, in-flight requests are dropped on SIGTERM.
func signalHandlingAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "signalhandling",
		Doc:      "flags main() that starts a server without signal handling for graceful shutdown",
		Run:      runSignalHandling,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

// serverMethodNames lists method names on receivers that indicate a
// server/listener is being started (e.g. srv.ListenAndServe()).
var serverMethodNames = map[string]bool{ //nolint:gochecknoglobals // package-internal lookup table, not mutable state
	"ListenAndServe":    true,
	"ListenAndServeTLS": true,
	"Serve":             true,
}

func runSignalHandling(pass *analysis.Pass) (any, error) {
	// Only relevant for package main where main() runs as an entry point.
	if pass.Pkg.Name() != "main" {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		funcDecl, isFuncDecl := node.(*ast.FuncDecl)
		if !isFuncDecl || funcDecl.Name.Name != "main" || funcDecl.Body == nil {
			return
		}

		var hasServer bool

		var hasSignal bool

		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
			call, isCall := n.(*ast.CallExpr)
			if !isCall {
				return true
			}

			// Check package-level function calls: http.ListenAndServe, net.Listen
			if isCallTo(call, pass.TypesInfo, httpPkgPath, "ListenAndServe") ||
				isCallTo(call, pass.TypesInfo, "net", "Listen") {
				hasServer = true
			}

			// Check method calls on receivers: srv.ListenAndServe(), srv.Serve(), etc.
			if sel, isSel := call.Fun.(*ast.SelectorExpr); isSel {
				if serverMethodNames[sel.Sel.Name] {
					hasServer = true
				}
			}

			// Check for signal handling calls.
			if isCallTo(call, pass.TypesInfo, "os/signal", "NotifyContext") ||
				isCallTo(call, pass.TypesInfo, "os/signal", "Notify") {
				hasSignal = true
			}

			return true
		})

		if hasServer && !hasSignal {
			pass.Reportf(funcDecl.Pos(), "main() starts a server without signal handling; use signal.NotifyContext for graceful shutdown")
		}
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
