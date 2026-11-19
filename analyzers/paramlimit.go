package analyzers

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// maxDomainParams is the largest number of counted parameters a signature
// may declare before the rule asks for a request struct or functional options.
const maxDomainParams = 3

// The logger and option exemptions match on these type-name suffixes rather
// than a list of packages.
const (
	loggerTypeSuffix = "Logger"
	optionTypeSuffix = "Option"
)

// paramLimitAnalyzer returns an analyzer that flags functions, methods,
// interface methods, and func-literal-typed struct fields with more than
// maxDomainParams domain parameters. Contexts, one logger, testing handles,
// func parameters, and functional options don't count, so a callback carrying
// domain data goes uncounted. A signature an outside interface or func type
// fixes is still reported; silence it with //nolint:paramlimit.
func paramLimitAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "paramlimit",
		Doc: fmt.Sprintf("flags signatures with more than %d domain parameters; use a request or config struct, or functional options",
			maxDomainParams),
		Run: runParamLimit,
	}
}

func runParamLimit(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		// Mocks mirror the signatures of the interfaces they stand in for.
		if isMockPath(pass.Fset.Position(file.Pos()).Filename) {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.FuncDecl:
				checkParamLimit(pass, typed.Name, typed.Type)
			case *ast.InterfaceType:
				checkFuncFields(pass, typed.Methods)
			case *ast.StructType:
				checkFuncFields(pass, typed.Fields)
			}

			return true
		})
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkFuncFields checks each named field whose type is a func type literal.
// That covers interface methods and struct fields such as
// `DownloadFunc func(...) error`, and skips embedded types and fields of a
// named func type.
func checkFuncFields(pass *analysis.Pass, fields *ast.FieldList) {
	if fields == nil {
		return
	}

	for _, field := range fields.List {
		funcType, isFunc := field.Type.(*ast.FuncType)
		if !isFunc {
			continue
		}

		for _, name := range field.Names {
			checkParamLimit(pass, name, funcType)
		}
	}
}

func checkParamLimit(pass *analysis.Pass, name *ast.Ident, funcType *ast.FuncType) {
	count := countDomainParams(pass.TypesInfo, funcType.Params)
	if count <= maxDomainParams {
		return
	}

	pass.Reportf(name.Pos(),
		"%s has %d domain parameters, more than the limit of %d; group them in a request or config struct, or use functional options",
		name.Name, count, maxDomainParams)
}

// countDomainParams counts each parameter name, or one per unnamed parameter,
// leaving out the parameters the rule exempts. Only the first logger is
// exempt, so a second one counts.
func countDomainParams(info *types.Info, params *ast.FieldList) int {
	if params == nil {
		return 0
	}

	var (
		count      int
		loggerSeen bool
	)

	for _, field := range params.List {
		nameCount := max(len(field.Names), 1)
		if isExemptParam(info, field) {
			continue
		}

		if !loggerSeen && hasNameSuffix(info.TypeOf(field.Type), loggerTypeSuffix) {
			loggerSeen = true
			nameCount--
		}

		count += nameCount
	}

	return count
}

// isExemptParam reports whether every name in field is exempt by type: a
// context, a testing handle, a func, or a variadic of functional options,
// whose element is a func or a type whose name ends in Option, such as
// gRPC's `...CallOption`. A variadic's recorded type is its slice, so the
// element type decides.
func isExemptParam(info *types.Info, field *ast.Field) bool {
	if ellipsis, isVariadic := field.Type.(*ast.Ellipsis); isVariadic {
		elem := info.TypeOf(ellipsis.Elt)

		return isFuncType(elem) || hasNameSuffix(elem, optionTypeSuffix)
	}

	typ := info.TypeOf(field.Type)

	// Unalias lets `type Ctx = context.Context` match. A renamed import
	// already names the same type, so it needs nothing extra.
	isContext := isNamedType(types.Unalias(typ), contextPkgPath, contextTypeName)

	return isContext || isTestingType(typ) || isFuncType(typ)
}

// isFuncType reports whether typ's underlying type is a func signature, which
// includes named func types such as `type CleanupFunc func()`.
func isFuncType(typ types.Type) bool {
	if typ == nil {
		return false
	}

	_, isSignature := typ.Underlying().(*types.Signature)

	return isSignature
}

// hasNameSuffix reports whether typ, after one pointer, is a named type whose
// name ends in suffix. Matching on the name covers slog, zap, logr, gRPC, and
// a project's own types without listing each package.
func hasNameSuffix(typ types.Type, suffix string) bool {
	named := namedTypeOf(typ)

	return named != nil && strings.HasSuffix(named.Obj().Name(), suffix)
}
