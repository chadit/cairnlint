package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// deepEqualArgCount is the arity of reflect.DeepEqual. Type-checked code
// always matches it; the guard keeps a malformed AST from panicking.
const deepEqualArgCount = 2

// reflectDeepEqualScalarAnalyzer returns an analyzer that flags
// reflect.DeepEqual calls on scalar operands. For identical types the call is
// a slow ==, boxing each operand onto the heap. For differing types it is
// always false, hiding a mismatch == would refuse to compile. Interface and
// type-parameter operands are skipped, since their runtime type is unknown.
func reflectDeepEqualScalarAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "reflectdeepequalscalar",
		Doc:      "flags reflect.DeepEqual on scalar operands; use == for identical types, and differing types are never equal",
		Run:      runReflectDeepEqualScalar,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func runReflectDeepEqualScalar(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		call, isCall := node.(*ast.CallExpr)
		if !isCall || len(call.Args) != deepEqualArgCount {
			return
		}

		if !isCallTo(call, pass.TypesInfo, reflectPkgPath, "DeepEqual") {
			return
		}

		left := pass.TypesInfo.TypeOf(call.Args[0])
		right := pass.TypesInfo.TypeOf(call.Args[1])

		if left == nil || right == nil || types.IsInterface(left) || types.IsInterface(right) {
			return
		}

		leftScalar := isScalarType(left)
		rightScalar := isScalarType(right)

		if !leftScalar && !rightScalar {
			return
		}

		if !types.Identical(left, right) {
			pass.Reportf(
				call.Pos(),
				"reflect.DeepEqual on %s and %s is always false because DeepEqual never equates distinct types; make the operand types match",
				left, right,
			)

			return
		}

		pass.Reportf(
			call.Pos(),
			"reflect.DeepEqual on %s operands; use == or != for scalar types",
			left,
		)
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isScalarType reports whether t's underlying type is a boolean, numeric,
// or string basic type, so == is defined and cheaper than reflection.
// Named types over those kinds count. Untyped kinds (including untyped
// nil), unsafe.Pointer, and invalid types do not.
func isScalarType(t types.Type) bool {
	basic, isBasic := t.Underlying().(*types.Basic)
	if !isBasic {
		return false
	}

	info := basic.Info()
	if info&types.IsUntyped != 0 {
		return false
	}

	return info&(types.IsBoolean|types.IsNumeric|types.IsString) != 0
}
