package analyzers

import (
	"go/ast"
	"go/types"
	"reflect"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

// factFreeSSAAnalyzer produces what [buildssa.Analyzer] does without requiring
// ctrlflow, whose FactTypes switch the multichecker to packages.LoadAllSyntax
// for the whole run. The cost is ctrlflow's noreturn data, so blocks after a
// call such as os.Exit stay in the graph. It is one shared value so the
// checker builds SSA once per package for every analyzer requiring it.
var factFreeSSAAnalyzer = &analysis.Analyzer{ //nolint:gochecknoglobals // shared so the checker runs it once per package
	Name:       "factfreessa",
	Doc:        "build SSA-form IR like buildssa, without the ctrlflow facts it requires",
	Run:        runFactFreeSSA,
	ResultType: reflect.TypeFor[*buildssa.SSA](),
}

func runFactFreeSSA(pass *analysis.Pass) (any, error) {
	// The analysis API has no place to share a Program across packages, so
	// each pass builds its own, the same way buildssa does.
	prog := ssa.NewProgram(pass.Fset, ssa.BuilderMode(0))

	for _, imported := range pass.Pkg.Imports() {
		prog.CreatePackage(imported, nil, nil, true)
	}

	ssaPkg := prog.CreatePackage(pass.Pkg, pass.Files, pass.TypesInfo, false)
	ssaPkg.Build()

	var funcs []*ssa.Function
	for _, file := range pass.Files {
		funcs = appendFileFuncs(funcs, prog, pass.TypesInfo, file)
	}

	return &buildssa.SSA{Pkg: ssaPkg, SrcFuncs: funcs}, nil
}

// appendFileFuncs appends the SSA function for each func declaration in file,
// each followed by its closures, which keeps SrcFuncs in source order.
func appendFileFuncs(funcs []*ssa.Function, prog *ssa.Program, info *types.Info, file *ast.File) []*ssa.Function {
	for _, decl := range file.Decls {
		funcDecl, isFuncDecl := decl.(*ast.FuncDecl)
		if !isFuncDecl {
			continue
		}

		declared, isFunc := info.Defs[funcDecl.Name].(*types.Func)
		if !isFunc {
			continue
		}

		funcs = appendWithAnonFuncs(funcs, prog.FuncValue(declared))
	}

	return funcs
}

func appendWithAnonFuncs(funcs []*ssa.Function, function *ssa.Function) []*ssa.Function {
	if function == nil {
		return funcs
	}

	funcs = append(funcs, function)
	for _, anon := range function.AnonFuncs {
		funcs = appendWithAnonFuncs(funcs, anon)
	}

	return funcs
}
