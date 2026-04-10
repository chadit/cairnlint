package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// mapPreallocAnalyzer returns an analyzer that flags maps created without a
// capacity hint that are subsequently populated inside a range loop. Without
// pre-allocation the runtime rehashes the map as it grows, wasting CPU and
// memory.
//
// The -min-range-len flag (default 8) suppresses diagnostics when the range
// source is a literal with fewer than that many elements. Maps <= 8 entries
// fit in one bucket and may be stack-allocated, so pre-allocation offers no
// benefit and can force a heap allocation.
func mapPreallocAnalyzer() *analysis.Analyzer {
	var minRangeLen int

	analyzer := &analysis.Analyzer{
		Name:     "mapprealloc",
		Doc:      "flags maps populated in range loops without a capacity hint; use make(map[K]V, len(source)) to pre-allocate",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}

	analyzer.Flags.IntVar(&minRangeLen, "min-range-len", defaultMinRangeLen,
		"minimum range-source length to trigger diagnostic; literals below this size are skipped")

	analyzer.Run = func(pass *analysis.Pass) (any, error) {
		return runMapPrealloc(pass, minRangeLen)
	}

	return analyzer
}

func runMapPrealloc(pass *analysis.Pass, minRangeLen int) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.BlockStmt)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		block, isBlock := node.(*ast.BlockStmt)
		if !isBlock {
			return
		}

		checkBlockForMapPrealloc(block.List, pass, minRangeLen)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkBlockForMapPrealloc scans statements in a block looking for map
// creations (make without capacity or empty composite literals) followed by a
// range loop that assigns into the same map variable.
func checkBlockForMapPrealloc(stmts []ast.Stmt, pass *analysis.Pass, minRangeLen int) {
	for idx, stmt := range stmts {
		assignStmt, isAssign := stmt.(*ast.AssignStmt)
		if !isAssign {
			continue
		}

		if len(assignStmt.Lhs) != 1 || len(assignStmt.Rhs) != 1 {
			continue
		}

		mapName := identName(assignStmt.Lhs[0])
		if mapName == "" {
			continue
		}

		if !isUncappedMapCreation(assignStmt.Rhs[0]) {
			continue
		}

		// Scan forward for a range loop that writes into this map.
		for ahead := idx + 1; ahead < len(stmts); ahead++ {
			rangeStmt, isRange := stmts[ahead].(*ast.RangeStmt)
			if !isRange {
				continue
			}

			if rangeBodyAssignsToMap(rangeStmt.Body, mapName) {
				// Small literal range source — skip but keep scanning for
				// subsequent range loops that may have a larger source.
				if litLen := rangeSourceLiteralLen(rangeStmt.X); litLen >= 0 && litLen < minRangeLen {
					continue
				}

				pass.Reportf(
					assignStmt.Pos(),
					"map %s populated in range loop without capacity hint; use make(map[K]V, len(source)) to pre-allocate",
					mapName,
				)

				break
			}
		}
	}
}

// isUncappedMapCreation reports whether expr is a map creation without a
// capacity hint. Matches two patterns:
//   - make(map[K]V) with exactly one arg (the type)
//   - map[K]V{} composite literal with no elements
func isUncappedMapCreation(expr ast.Expr) bool {
	switch node := expr.(type) {
	case *ast.CallExpr:
		ident, isIdent := node.Fun.(*ast.Ident)
		if !isIdent || ident.Name != "make" {
			return false
		}

		// make(map[K]V) has 1 arg, make(map[K]V, cap) has 2.
		if len(node.Args) != 1 {
			return false
		}

		_, isMap := node.Args[0].(*ast.MapType)

		return isMap

	case *ast.CompositeLit:
		_, isMap := node.Type.(*ast.MapType)

		return isMap && len(node.Elts) == 0
	}

	return false
}

// rangeBodyAssignsToMap reports whether the range body contains an assignment
// of the form mapVar[...] = ... for the given map variable name. Does not
// descend into function literals since those run in a different scope.
func rangeBodyAssignsToMap(body *ast.BlockStmt, mapName string) bool {
	if body == nil {
		return false
	}

	var found bool

	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil || found {
			return false
		}

		// Don't descend into closures; they capture the map but are a
		// different execution scope and may not run synchronously.
		if _, isFunc := node.(*ast.FuncLit); isFunc {
			return false
		}

		assign, isAssign := node.(*ast.AssignStmt)
		if !isAssign {
			return true
		}

		for _, lhs := range assign.Lhs {
			indexExpr, isIndex := lhs.(*ast.IndexExpr)
			if !isIndex {
				continue
			}

			if identName(indexExpr.X) == mapName {
				found = true

				return false
			}
		}

		return true
	})

	return found
}

// identName returns the name string for a simple *ast.Ident, or "" for
// anything else.
func identName(expr ast.Expr) string {
	ident, isIdent := expr.(*ast.Ident)
	if !isIdent {
		return ""
	}

	return ident.Name
}
