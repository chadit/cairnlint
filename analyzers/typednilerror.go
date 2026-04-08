package analyzers

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

// typedNilErrorAnalyzer returns an analyzer that flags returning a typed nil
// pointer as an error interface. In Go, a nil pointer wrapped in an interface
// produces a non-nil interface value, so callers checking `if err != nil` get
// a true result even though the underlying pointer is nil. This is a well-known
// gotcha that causes subtle bugs.
func typedNilErrorAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "typednilerror",
		Doc:      "flags returning a typed nil pointer as an error interface, which produces a non-nil error value",
		Run:      runTypedNilError,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	}
}

// errorType caches the universe-scope named error type for comparison.
var errorType = types.Universe.Lookup("error").Type() //nolint:gochecknoglobals // package-internal type constant

func runTypedNilError(pass *analysis.Pass) (any, error) {
	ssaResult, castOK := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	for _, ssaFunc := range ssaResult.SrcFuncs {
		checkFunctionForTypedNilError(pass, ssaFunc)
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkFunctionForTypedNilError inspects a single SSA function for return
// statements that wrap a typed nil pointer into the error interface.
func checkFunctionForTypedNilError(pass *analysis.Pass, ssaFunc *ssa.Function) {
	sig := ssaFunc.Signature

	results := sig.Results()
	if results == nil {
		return
	}

	// Find which result indices are the error interface type.
	var errorIndices []int

	for i := range results.Len() {
		if types.Identical(results.At(i).Type(), errorType) {
			errorIndices = append(errorIndices, i)
		}
	}

	if len(errorIndices) == 0 {
		return
	}

	for _, block := range ssaFunc.Blocks {
		for _, instr := range block.Instrs {
			ret, isReturn := instr.(*ssa.Return)
			if !isReturn {
				continue
			}

			for _, idx := range errorIndices {
				if idx >= len(ret.Results) {
					continue
				}

				checkReturnOperand(pass, ret.Results[idx], ret.Pos())
			}
		}
	}
}

// checkReturnOperand examines a single returned SSA value. In SSA, returning a
// typed nil as error becomes:
//
//	t0 = make interface error <- *MyError (nil:*MyError)
//	return t0
//
// We look for MakeInterface whose underlying operand (X) is a pointer type and
// is a nil constant. The retPos parameter provides a fallback source position
// from the return statement, since synthetic MakeInterface instructions often
// lack a source position.
func checkReturnOperand(pass *analysis.Pass, val ssa.Value, retPos token.Pos) {
	makeIface, isMakeInterface := val.(*ssa.MakeInterface)
	if !isMakeInterface {
		return
	}

	innerVal := makeIface.X

	// The underlying type must be a pointer to catch typed nils.
	if _, isPtr := innerVal.Type().Underlying().(*types.Pointer); !isPtr {
		return
	}

	// Check for nil constant: covers `return (*MyError)(nil)` and
	// `var err *MyError; return err` (SSA optimizes zero-value locals to nil const).
	constVal, isConst := innerVal.(*ssa.Const)
	if !isConst {
		return
	}

	if !constVal.IsNil() {
		return
	}

	// Prefer the MakeInterface position when available (explicit typed nil cast),
	// fall back to the return statement position (zero-value variable case).
	pos := makeIface.Pos()
	if !pos.IsValid() {
		pos = retPos
	}

	pass.Reportf(
		pos,
		"returning typed nil as error interface produces non-nil error; return explicit nil instead",
	)
}
