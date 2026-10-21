package analyzers

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Argument positions in make(T, length, capacity).
const (
	makeLengthArg   = 1
	makeCapacityArg = 2
)

// literalSizeAnalyzer returns an analyzer that flags a bare numeric literal
// used as a size: an array length, a channel capacity, and in test files a
// slice length or capacity or a map size hint. It covers the shapes mnd
// skips, so every value counts, 0 and 1 included.
func literalSizeAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "literalsize",
		Doc:      "flags a bare numeric literal used as an array length, channel capacity, or (in tests) make size; name it as a constant",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      runLiteralSize,
	}
}

func runLiteralSize(pass *analysis.Pass) (any, error) {
	insp, castOK := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	nodeFilter := []ast.Node{
		(*ast.ArrayType)(nil),
		(*ast.CallExpr)(nil),
	}

	insp.WithStack(nodeFilter, func(node ast.Node, push bool, stack []ast.Node) bool {
		if !push || inFileScopeValueDecl(stack) {
			return true
		}

		switch typed := node.(type) {
		case *ast.ArrayType:
			checkArrayLength(pass, typed)
		case *ast.CallExpr:
			checkMakeSize(pass, typed)
		}

		return true
	})

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// inFileScopeValueDecl reports whether the stack sits inside a file-scope
// const or var declaration. That declaration is where a named value lives,
// so its literals are the fix, not the problem; stringer emits var x [1]struct{}.
// The skip ends at a function literal, whose body is ordinary code.
func inFileScopeValueDecl(stack []ast.Node) bool {
	const fileDeclDepth = 1 // stack[0] is the *ast.File

	if len(stack) <= fileDeclDepth {
		return false
	}

	genDecl, isGen := stack[fileDeclDepth].(*ast.GenDecl)
	if !isGen || (genDecl.Tok != token.CONST && genDecl.Tok != token.VAR) {
		return false
	}

	for _, node := range stack[fileDeclDepth+1:] {
		if _, isFuncLit := node.(*ast.FuncLit); isFuncLit {
			return false
		}
	}

	return true
}

// checkArrayLength reports a literal array length. [0]T is left alone
// because it only marks a struct as not comparable.
func checkArrayLength(pass *analysis.Pass, array *ast.ArrayType) {
	lit, isLit := array.Len.(*ast.BasicLit)
	if !isLit || isZeroLiteral(pass, lit) {
		return
	}

	reportLiteralSize(pass, lit, "array length")
}

// checkMakeSize reports literal size arguments to the builtin make. Slice
// and map sizes are reported only in test files, because mnd already
// reports a non-0/1 argument outside tests.
func checkMakeSize(pass *analysis.Pass, call *ast.CallExpr) {
	if len(call.Args) <= makeLengthArg || !isBuiltinMake(pass, call) {
		return
	}

	argType := pass.TypesInfo.TypeOf(call.Args[0])
	if argType == nil {
		return
	}

	switch coreType(argType).(type) {
	case *types.Chan:
		reportIfLiteral(pass, call.Args[makeLengthArg], "channel capacity")
	case *types.Slice:
		reportSliceSizes(pass, call)
	case *types.Map:
		if isTestFile(pass, call) {
			reportIfLiteral(pass, call.Args[makeLengthArg], "make length")
		}
	}
}

// reportSliceSizes reports a literal length and capacity of make on a
// slice type in a test file.
func reportSliceSizes(pass *analysis.Pass, call *ast.CallExpr) {
	if !isTestFile(pass, call) {
		return
	}

	reportIfLiteral(pass, call.Args[makeLengthArg], "make length")

	if len(call.Args) > makeCapacityArg {
		reportIfLiteral(pass, call.Args[makeCapacityArg], "make capacity")
	}
}

// coreType returns the underlying type make builds. For a type parameter
// the first constraint term stands in, because make only compiles when
// every term shares one core type.
func coreType(typ types.Type) types.Type {
	typeParam, isParam := typ.(*types.TypeParam)
	if !isParam {
		return typ.Underlying()
	}

	iface, isIface := typeParam.Constraint().Underlying().(*types.Interface)
	if !isIface {
		return nil
	}

	for embedded := range iface.EmbeddedTypes() {
		if union, isUnion := embedded.(*types.Union); isUnion {
			return union.Term(0).Type().Underlying()
		}

		if _, isIface := embedded.Underlying().(*types.Interface); !isIface {
			return embedded.Underlying()
		}
	}

	return nil
}

// isBuiltinMake reports whether call invokes the predeclared make, so a
// user-defined function named make is ignored.
func isBuiltinMake(pass *analysis.Pass, call *ast.CallExpr) bool {
	ident, isIdent := call.Fun.(*ast.Ident)
	if !isIdent {
		return false
	}

	builtin, isBuiltin := pass.TypesInfo.ObjectOf(ident).(*types.Builtin)

	return isBuiltin && builtin.Name() == "make"
}

// isZeroLiteral reports whether lit evaluates to zero, which covers 0x0
// and 00 as well as 0.
func isZeroLiteral(pass *analysis.Pass, lit *ast.BasicLit) bool {
	value := pass.TypesInfo.Types[lit].Value

	return value != nil && constant.Sign(value) == 0
}

func reportIfLiteral(pass *analysis.Pass, expr ast.Expr, where string) {
	lit, isLit := expr.(*ast.BasicLit)
	if !isLit {
		return
	}

	reportLiteralSize(pass, lit, where)
}

func reportLiteralSize(pass *analysis.Pass, lit *ast.BasicLit, where string) {
	pass.Reportf(lit.Pos(), "literal size in %s; name it as a constant", where)
}
