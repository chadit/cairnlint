package analyzers

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// lastIndexFunc is the function whose result-then-slice pattern CutLast
// collapses into a single call.
const lastIndexFunc = "LastIndex"

// preferCutLastAnalyzer returns an analyzer that flags a LastIndex result used
// to slice the same operand. Go 1.27 added strings.CutLast and bytes.CutLast,
// which return the two halves and a found flag in one call.
//
// The manual form has to get the offset arithmetic right by hand: the tail
// starts at idx+len(sep), and forgetting the separator length silently leaves
// it on the front of the result. CutLast removes that arithmetic entirely.
func preferCutLastAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "prefercutlast",
		Doc:      "flags strings/bytes.LastIndex whose result slices the same operand; use CutLast instead (Go 1.27+)",
		Run:      runPreferCutLast,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

// cutLastCandidate records a LastIndex call along with the operand it indexes
// into, so a later slice expression can be matched against both.
type cutLastCandidate struct {
	call    *ast.CallExpr
	operand types.Object
	pkgName string
}

func runPreferCutLast(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	// Nested function literals are visited both on their own and as part of the
	// enclosing body, so positions already reported are skipped.
	reported := make(map[token.Pos]struct{})

	insp.Preorder(nodeFilter, func(node ast.Node) {
		body := funcBody(node)
		if body == nil {
			return
		}

		for index, candidate := range collectCutLastCandidates(body, pass.TypesInfo) {
			if _, done := reported[candidate.call.Pos()]; done {
				continue
			}

			if !slicesOperandWithIndex(body, pass.TypesInfo, candidate.operand, index) {
				continue
			}

			if !goVersionAtLeast(pass, candidate.call.Pos(), goVersion127) {
				continue
			}

			reported[candidate.call.Pos()] = struct{}{}

			pass.Reportf(candidate.call.Pos(),
				"%[1]s.LastIndex followed by slicing can be replaced with %[1]s.CutLast (Go 1.27)", candidate.pkgName)
		}
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// funcBody returns the body block of a FuncDecl or FuncLit node.
func funcBody(node ast.Node) *ast.BlockStmt {
	switch fn := node.(type) {
	case *ast.FuncDecl:
		return fn.Body
	case *ast.FuncLit:
		return fn.Body
	default:
		return nil
	}
}

// collectCutLastCandidates finds every `idx := strings.LastIndex(s, sep)` in
// body, keyed by the object bound to the result.
//
// Only a plain identifier operand is tracked. Matching an arbitrary expression
// would mean comparing subtrees for equality, and the identifier case covers
// the shape CutLast was added to replace.
func collectCutLastCandidates(body *ast.BlockStmt, info *types.Info) map[types.Object]cutLastCandidate {
	candidates := make(map[types.Object]cutLastCandidate)

	ast.Inspect(body, func(node ast.Node) bool {
		assign, isAssign := node.(*ast.AssignStmt)
		if !isAssign || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return true
		}

		call, isCall := assign.Rhs[0].(*ast.CallExpr)
		if !isCall || len(call.Args) != 2 {
			return true
		}

		pkgName := lastIndexPackage(call, info)
		if pkgName == "" {
			return true
		}

		operand, hasOperand := identObject(call.Args[0], info)
		if !hasOperand {
			return true
		}

		result, hasResult := identObject(assign.Lhs[0], info)
		if !hasResult {
			return true
		}

		candidates[result] = cutLastCandidate{call: call, operand: operand, pkgName: pkgName}

		return true
	})

	return candidates
}

// lastIndexPackage returns "strings" or "bytes" when call is that package's
// LastIndex function, and the empty string otherwise.
func lastIndexPackage(call *ast.CallExpr, info *types.Info) string {
	if isCallTo(call, info, stringsPkgPath, lastIndexFunc) {
		return stringsPkgPath
	}

	if isCallTo(call, info, bytesPkgPath, lastIndexFunc) {
		return bytesPkgPath
	}

	return ""
}

// slicesOperandWithIndex reports whether body slices operand using a bound that
// mentions index, which is the half of the pattern CutLast replaces.
func slicesOperandWithIndex(body *ast.BlockStmt, info *types.Info, operand, index types.Object) bool {
	var found bool

	ast.Inspect(body, func(node ast.Node) bool {
		if found {
			return false
		}

		slice, isSlice := node.(*ast.SliceExpr)
		if !isSlice {
			return true
		}

		if sliced, isIdent := identObject(slice.X, info); !isIdent || sliced != operand {
			return true
		}

		found = mentionsObject(slice.Low, info, index) || mentionsObject(slice.High, info, index)

		return !found
	})

	return found
}

// mentionsObject reports whether expr contains an identifier resolving to obj.
// The bound is an expression rather than a bare identifier because the tail
// half of the pattern is written as idx+len(sep).
func mentionsObject(expr ast.Expr, info *types.Info, obj types.Object) bool {
	if expr == nil {
		return false
	}

	var found bool

	ast.Inspect(expr, func(node ast.Node) bool {
		if found {
			return false
		}

		if ident, isIdent := node.(*ast.Ident); isIdent && info.ObjectOf(ident) == obj {
			found = true
		}

		return !found
	})

	return found
}
