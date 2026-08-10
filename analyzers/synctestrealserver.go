package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// httptestRealServerCtors are the httptest constructors backed by a real
// listener on the loopback interface.
var httptestRealServerCtors = []string{"NewServer", "NewTLSServer", "NewUnstartedServer"} //nolint:gochecknoglobals // package-internal lookup table

// synctestRealServerAnalyzer returns an analyzer that flags real httptest
// servers created inside a synctest.Test closure. Go 1.27 added
// httptest.NewTestServer, which is backed by an in-memory network built for
// synctest.
//
// A real listener puts the socket outside the bubble, so a goroutine waiting on
// it never counts as durably blocked and synctest.Wait can hang or return
// early. The failure shows up as a flaky test far from this line, which is why
// it is worth flagging at the constructor.
func synctestRealServerAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "synctestrealserver",
		Doc:      "flags httptest.NewServer and friends inside synctest.Test; use httptest.NewTestServer instead (Go 1.27+)",
		Run:      runSynctestRealServer,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runSynctestRealServer(pass *analysis.Pass) (any, error) {
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

		ctor := realServerCtorName(call, pass.TypesInfo)
		if ctor == "" {
			return true
		}

		if !isInsideSynctestClosure(stack, pass.TypesInfo) {
			return true
		}

		if !goVersionAtLeast(pass, call.Pos(), goVersion127) {
			return true
		}

		pass.Reportf(call.Pos(),
			"httptest.%s inside synctest.Test listens on a real socket that the bubble cannot see; use httptest.NewTestServer (Go 1.27)",
			ctor)

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// realServerCtorName returns the httptest constructor name when call creates a
// server backed by a real listener, and the empty string otherwise.
func realServerCtorName(call *ast.CallExpr, info *types.Info) string {
	for _, ctor := range httptestRealServerCtors {
		if isCallTo(call, info, httptestPkgPath, ctor) {
			return ctor
		}
	}

	return ""
}
