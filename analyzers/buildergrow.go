package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// builderWriteMethods lists the strings.Builder methods that append data and
// benefit from a preceding Grow call when used inside a loop.
var builderWriteMethods = map[string]bool{ //nolint:gochecknoglobals // package-internal lookup table, not mutable state
	"Write":       true,
	"WriteString": true,
	"WriteByte":   true,
	"WriteRune":   true,
}

// builderGrowAnalyzer returns an analyzer that flags strings.Builder write
// methods called inside loops without a preceding Grow. Without pre-allocation,
// each write may trigger a reallocation, turning O(n) appends into O(n^2).
//
// The -min-range-len flag (default 8) suppresses diagnostics when the range
// source is a literal with fewer than that many elements. Builder's doubling
// strategy handles small iteration counts in 2-3 allocs.
func builderGrowAnalyzer() *analysis.Analyzer {
	var minRangeLen int

	analyzer := &analysis.Analyzer{
		Name:     "buildergrow",
		Doc:      "flags strings.Builder write methods inside loops without a preceding Grow(); without pre-allocation the builder reallocates on each write, causing O(n^2) behavior",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}

	analyzer.Flags.IntVar(&minRangeLen, "min-range-len", defaultMinRangeLen,
		"minimum range-source length to trigger diagnostic; literals below this size are skipped")

	analyzer.Run = func(pass *analysis.Pass) (any, error) {
		return runBuilderGrow(pass, minRangeLen)
	}

	return analyzer
}

func runBuilderGrow(pass *analysis.Pass, minRangeLen int) (any, error) {
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

		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel {
			return true
		}

		methodName := sel.Sel.Name
		if !builderWriteMethods[methodName] {
			return true
		}

		if !isBuilderReceiver(sel.X, pass.TypesInfo) {
			return true
		}

		if !isInsideLoop(stack) {
			return true
		}

		loopNode := enclosingLoop(stack)
		if loopNode == nil {
			return true
		}

		receiver := receiverIdent(sel.X)
		if receiver == "" {
			return true
		}

		if hasGrowBeforeLoop(stack, loopNode, receiver, pass.TypesInfo) {
			return true
		}

		// Builder declared inside the loop body is scoped to a single
		// iteration, so cross-iteration accumulation is impossible and Grow is pointless.
		if isDeclaredInsideLoop(loopNode, receiver, pass.TypesInfo) {
			return true
		}

		// For range loops over small literal sources, skip the diagnostic.
		if rangeLoop, isRange := loopNode.(*ast.RangeStmt); isRange {
			if litLen := rangeSourceLiteralLen(rangeLoop.X); litLen >= 0 && litLen < minRangeLen {
				return true
			}
		}

		pass.Reportf(
			call.Pos(),
			"strings.Builder.%s() in loop without Grow; pre-allocate with Grow() to avoid repeated reallocations",
			methodName,
		)

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// isBuilderReceiver reports whether expr resolves to a *strings.Builder or
// strings.Builder type.
func isBuilderReceiver(expr ast.Expr, info *types.Info) bool {
	recvType := info.TypeOf(expr)
	if recvType == nil {
		return false
	}

	// Unwrap pointer if present.
	if ptr, isPtr := recvType.(*types.Pointer); isPtr {
		recvType = ptr.Elem()
	}

	named, isNamed := recvType.(*types.Named)
	if !isNamed {
		return false
	}

	obj := named.Obj()

	return obj.Pkg() != nil && obj.Pkg().Path() == "strings" && obj.Name() == "Builder"
}

// enclosingLoop walks the inspector stack backwards and returns the nearest
// enclosing loop node (ForStmt or RangeStmt). Stops at function boundaries.
func enclosingLoop(stack []ast.Node) ast.Node {
	for idx := len(stack) - 2; idx >= 0; idx-- {
		switch stack[idx].(type) {
		case *ast.FuncLit, *ast.FuncDecl:
			return nil
		case *ast.ForStmt, *ast.RangeStmt:
			return stack[idx]
		}
	}

	return nil
}

// hasGrowBeforeLoop checks whether a Grow() call on the same receiver appears
// in the enclosing block before the loop node. Walks the inspector stack to
// find the block that directly contains the loop, then scans statements
// preceding the loop for a matching Grow call.
func hasGrowBeforeLoop(stack []ast.Node, loopNode ast.Node, receiver string, info *types.Info) bool {
	// Find the block statement that contains the loop by walking the stack.
	var enclosingBlock *ast.BlockStmt

	for idx := len(stack) - 2; idx >= 0; idx-- {
		if stack[idx] == loopNode {
			// The parent of the loop in the stack should be a block or similar container.
			for parent := idx - 1; parent >= 0; parent-- {
				if block, isBlock := stack[parent].(*ast.BlockStmt); isBlock {
					enclosingBlock = block

					break
				}
			}

			break
		}
	}

	if enclosingBlock == nil {
		return false
	}

	// Walk statements in the block. Once we reach the loop, stop.
	// Any Grow on the same receiver before the loop counts.
	for _, stmt := range enclosingBlock.List {
		if stmtContainsNode(stmt, loopNode) {
			break
		}

		if stmtHasGrowCall(stmt, receiver, info) {
			return true
		}
	}

	return false
}

// stmtContainsNode reports whether stmt is or contains target.
func stmtContainsNode(stmt ast.Stmt, target ast.Node) bool {
	if stmt == target {
		return true
	}

	var found bool

	ast.Inspect(stmt, func(node ast.Node) bool {
		if node == target {
			found = true

			return false
		}

		return !found
	})

	return found
}

// isDeclaredInsideLoop reports whether a variable with the given receiver
// name is declared inside the loop body as a strings.Builder. A builder
// scoped to a single iteration has no cross-iteration accumulation, so
// flagging it for a missing Grow is a false positive.
func isDeclaredInsideLoop(loopNode ast.Node, receiver string, info *types.Info) bool {
	var body *ast.BlockStmt

	switch loop := loopNode.(type) {
	case *ast.RangeStmt:
		body = loop.Body
	case *ast.ForStmt:
		body = loop.Body
	default:
		return false
	}

	if body == nil {
		return false
	}

	for _, stmt := range body.List {
		switch decl := stmt.(type) {
		case *ast.DeclStmt:
			if varDeclDeclaresBuilder(decl, receiver, info) {
				return true
			}

		case *ast.AssignStmt:
			if shortVarDeclaresBuilder(decl, receiver, info) {
				return true
			}
		}
	}

	return false
}

// varDeclDeclaresBuilder reports whether a DeclStmt contains a var
// declaration of the given name typed as strings.Builder.
func varDeclDeclaresBuilder(decl *ast.DeclStmt, receiver string, info *types.Info) bool {
	genDecl, isGen := decl.Decl.(*ast.GenDecl)
	if !isGen {
		return false
	}

	for _, spec := range genDecl.Specs {
		valueSpec, isValue := spec.(*ast.ValueSpec)
		if !isValue {
			continue
		}

		for _, name := range valueSpec.Names {
			if name.Name == receiver && isBuilderTypeExpr(valueSpec.Type, info) {
				return true
			}
		}
	}

	return false
}

// shortVarDeclaresBuilder reports whether a := assignment declares the
// given name with a strings.Builder type on the right-hand side.
func shortVarDeclaresBuilder(assign *ast.AssignStmt, receiver string, info *types.Info) bool {
	if assign.Tok.String() != ":=" {
		return false
	}

	for idx, lhs := range assign.Lhs {
		ident, isIdent := lhs.(*ast.Ident)
		if !isIdent || ident.Name != receiver {
			continue
		}

		if idx < len(assign.Rhs) {
			rhsType := info.TypeOf(assign.Rhs[idx])
			if rhsType != nil && isBuilderType(rhsType) {
				return true
			}
		}
	}

	return false
}

// isBuilderTypeExpr checks whether a type expression refers to strings.Builder.
func isBuilderTypeExpr(expr ast.Expr, info *types.Info) bool {
	if expr == nil {
		return false
	}

	tv, hasType := info.Types[expr]
	if !hasType {
		return false
	}

	return isBuilderType(tv.Type)
}

// isBuilderType checks whether a type (including pointers) is strings.Builder.
func isBuilderType(t types.Type) bool {
	if ptr, isPtr := t.(*types.Pointer); isPtr {
		t = ptr.Elem()
	}

	named, isNamed := t.(*types.Named)
	if !isNamed {
		return false
	}

	obj := named.Obj()

	return obj.Pkg() != nil && obj.Pkg().Path() == "strings" && obj.Name() == "Builder"
}

// stmtHasGrowCall reports whether stmt contains a call to Grow() on a
// strings.Builder with the specified receiver identity.
func stmtHasGrowCall(stmt ast.Stmt, receiver string, info *types.Info) bool {
	var found bool

	ast.Inspect(stmt, func(node ast.Node) bool {
		if node == nil || found {
			return false
		}

		call, isCall := node.(*ast.CallExpr)
		if !isCall {
			return true
		}

		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel || sel.Sel.Name != "Grow" {
			return true
		}

		if !isBuilderReceiver(sel.X, info) {
			return true
		}

		if receiverIdent(sel.X) == receiver {
			found = true

			return false
		}

		return true
	})

	return found
}
