package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// tlsRandField is the tls.Config field deprecated in Go 1.27.
const tlsRandField = "Rand"

// tlsConfigType is the crypto/tls type carrying that field.
const tlsConfigType = "Config"

// tlsRandMessage explains the replacement for both the literal and the
// assignment form.
const tlsRandMessage = "tls.Config.Rand is deprecated as of Go 1.27; leave it nil to draw from crypto/rand, " +
	"or call testing/cryptotest.SetGlobalRandom in tests"

// tlsConfigRandAnalyzer returns an analyzer that flags writes to
// tls.Config.Rand, deprecated in Go 1.27.
//
// Supplying a custom source was the only way to make a handshake repeatable,
// and it silently weakens every handshake that config serves if the source is
// not what the author assumed. Go 1.27 moved that capability into
// testing/cryptotest, where it cannot reach production code by accident.
func tlsConfigRandAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "tlsconfigrand",
		Doc:      "flags writes to the tls.Config.Rand field, deprecated in Go 1.27",
		Run:      runTLSConfigRand,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runTLSConfigRand(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.CompositeLit)(nil),
		(*ast.AssignStmt)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		switch stmt := node.(type) {
		case *ast.CompositeLit:
			reportTLSRandInLiteral(pass, stmt)
		case *ast.AssignStmt:
			reportTLSRandAssign(pass, stmt)
		}
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// reportTLSRandInLiteral flags a Rand key inside a tls.Config literal.
func reportTLSRandInLiteral(pass *analysis.Pass, lit *ast.CompositeLit) {
	if !isNamedType(pass.TypesInfo.TypeOf(lit.Type), tlsPkgPath, tlsConfigType) {
		return
	}

	for _, elt := range lit.Elts {
		kv, isKV := elt.(*ast.KeyValueExpr)
		if !isKV {
			continue
		}

		key, isIdent := kv.Key.(*ast.Ident)
		if !isIdent || key.Name != tlsRandField {
			continue
		}

		if !goVersionAtLeast(pass, key.Pos(), goVersion127) {
			continue
		}

		pass.Reportf(key.Pos(), "%s", tlsRandMessage)
	}
}

// reportTLSRandAssign flags cfg.Rand = src on a tls.Config value or pointer.
func reportTLSRandAssign(pass *analysis.Pass, assign *ast.AssignStmt) {
	for _, lhs := range assign.Lhs {
		sel, isSel := lhs.(*ast.SelectorExpr)
		if !isSel || sel.Sel.Name != tlsRandField {
			continue
		}

		if !isTLSConfigExpr(sel.X, pass.TypesInfo) {
			continue
		}

		if !goVersionAtLeast(pass, sel.Pos(), goVersion127) {
			continue
		}

		pass.Reportf(sel.Pos(), "%s", tlsRandMessage)
	}
}

// isTLSConfigExpr reports whether expr is a tls.Config value or pointer to one.
func isTLSConfigExpr(expr ast.Expr, info *types.Info) bool {
	recv := info.TypeOf(expr)
	if ptr, isPtr := recv.(*types.Pointer); isPtr {
		recv = ptr.Elem()
	}

	return isNamedType(recv, tlsPkgPath, tlsConfigType)
}
