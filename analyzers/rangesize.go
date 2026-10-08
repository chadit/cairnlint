package analyzers

import (
	"go/ast"
	"go/token"
	"strconv"
)

// defaultMinRangeLen is the default threshold below which range-loop
// diagnostics are suppressed.  Maps <= 8 entries fit in one bucket and
// may be stack-allocated; Builder's doubling strategy handles <= 8
// iterations in 2-3 allocs.
const defaultMinRangeLen = 8

// rangeSourceLiteralLen returns the statically-known iteration count of
// expr when it is a composite literal, an integer literal (a Go 1.22 range
// over an integer), or a local variable assigned from either. Returns -1
// for any dynamic expression whose length cannot be determined at compile
// time.
func rangeSourceLiteralLen(expr ast.Expr) int {
	switch node := expr.(type) {
	case *ast.CompositeLit:
		return len(node.Elts)

	case *ast.BasicLit:
		if node.Kind != token.INT {
			return -1
		}

		count, err := strconv.Atoi(node.Value)
		if err != nil {
			return -1
		}

		return count

	case *ast.Ident:
		if node.Obj == nil || node.Obj.Decl == nil {
			return -1
		}

		// Resolve one level: var x = []T{...} or x := []T{...}
		switch decl := node.Obj.Decl.(type) {
		case *ast.ValueSpec:
			if len(decl.Values) == 1 {
				return rangeSourceLiteralLen(decl.Values[0])
			}
		case *ast.AssignStmt:
			if len(decl.Rhs) == 1 {
				return rangeSourceLiteralLen(decl.Rhs[0])
			}
		}

		return -1

	default:
		return -1
	}
}
