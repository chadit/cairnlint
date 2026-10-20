package analyzers

import (
	"go/ast"
	"go/token"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/astutil"
)

// defaultMinRangeLen is the default threshold below which range-loop
// diagnostics are suppressed.  Maps <= 8 entries fit in one bucket and
// may be stack-allocated; Builder's doubling strategy handles <= 8
// iterations in 2-3 allocs.
const defaultMinRangeLen = 8

// rangeSourceLiteralLen returns the statically-known iteration count of
// expr when it is a composite literal, an integer literal (a Go 1.22 range
// over an integer), or a variable declared in this package from either with
// a single-value var or := (one level, any file of the package). Returns -1
// for any dynamic expression whose length cannot be determined at compile
// time.
func rangeSourceLiteralLen(pass *analysis.Pass, expr ast.Expr) int {
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
		initializer := declaringInitializer(pass, node)
		if initializer == nil {
			return -1
		}

		return rangeSourceLiteralLen(pass, initializer)

	default:
		return -1
	}
}

// declaringInitializer resolves one level: for an ident declared as
// var x = expr or x := expr in this package, it returns expr. It returns nil
// for parameters, multi-value declarations, and objects outside the pass.
func declaringInitializer(pass *analysis.Pass, ident *ast.Ident) ast.Expr {
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return nil
	}

	file := fileContaining(pass, obj.Pos())
	if file == nil {
		return nil
	}

	// path[0] is the declaring ident and path[1] its parent declaration.
	path, _ := astutil.PathEnclosingInterval(file, obj.Pos(), obj.Pos())
	if len(path) <= 1 {
		return nil
	}

	switch decl := path[1].(type) {
	case *ast.ValueSpec:
		if len(decl.Values) == 1 {
			return decl.Values[0]
		}
	case *ast.AssignStmt:
		if len(decl.Rhs) == 1 {
			return decl.Rhs[0]
		}
	}

	return nil
}
