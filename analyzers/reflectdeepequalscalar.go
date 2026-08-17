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
// reflect.DeepEqual calls whose operands are scalars (numbers, strings,
// bools). For identical scalar types the call is a slow spelling of ==:
// DeepEqual leaks its any parameters, so each operand is boxed onto the
// heap (two allocations per call for any value the runtime cannot
// intern, meaning ints of 256 or more, floats, and non-empty strings),
// and the reflection walk runs 8x to 25x slower than the single compare
// instruction == compiles to.
// For differing concrete types the call is always false, because
// DeepEqual never equates values of distinct types, and == would have
// refused to compile, so the reflection call hides a type mismatch the
// compiler would otherwise have caught.
//
// Operands whose static type is an interface or a type parameter are
// skipped: the runtime type is unknown, so DeepEqual may be the only
// option (see sync/map_test.go and the recover() comparisons in
// encoding/json).
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
