package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
)

type declaredInterface struct {
	iface *types.Interface
	self  types.Type
	name  string
	pos   token.Pos
}

// agentLoneImplAnalyzer flags an interface that exactly one type in the same
// package implements. The concrete type can go straight into the signature,
// removing the interface and its call-site indirection. It is agent-only
// because the count is package-scoped: an implementation in another package is
// invisible here, so an LLM searches the module before acting.
func agentLoneImplAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "agentloneimpl",
		Doc:  "[agent] flags interfaces with exactly one implementing type in the same package",
		Run:  runAgentLoneImpl,
	}
}

func runAgentLoneImpl(pass *analysis.Pass) (any, error) {
	if skipUncountablePass(pass) {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	implementers := namedTypes(pass)

	for _, decl := range declaredInterfaces(pass) {
		reportLoneImpl(pass, decl, implementers)
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// skipUncountablePass drops passes whose per-package use count would be wrong
// or duplicated. An external test package declares nothing worth counting.
// With in-package _test.go files the plain variant is skipped, since the
// augmented variant also counts those files. External test files make no
// augmented variant, so the plain variant must report.
func skipUncountablePass(pass *analysis.Pass) bool {
	if pass.Pkg == nil || strings.HasSuffix(pass.Pkg.Name(), "_test") {
		return true
	}

	if hasTestFiles(pass) {
		return false
	}

	return packageDirHasInternalTests(pass)
}

// packageDirHasInternalTests reports whether the package's directory holds a
// _test.go file that belongs to the package itself rather than to its
// external _test package, and that the build includes. A file the build
// leaves out by tag or GOOS makes no test variant either.
func packageDirHasInternalTests(pass *analysis.Pass) bool {
	if len(pass.Files) == 0 {
		return false
	}

	dir := filepath.Dir(pass.Fset.Position(pass.Files[0].Pos()).Filename)

	matches, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		return false
	}

	return slices.ContainsFunc(matches, func(path string) bool {
		return !slices.Contains(pass.IgnoredFiles, path) && testFilePackage(path) == pass.Pkg.Name()
	})
}

// testFilePackage returns the package name a test file declares, or "" when
// its package clause cannot be read.
func testFilePackage(path string) string {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
	if err != nil {
		return ""
	}

	return file.Name.Name
}

func declaredInterfaces(pass *analysis.Pass) []declaredInterface {
	found := make([]declaredInterface, 0, len(pass.Files))

	for _, file := range pass.Files {
		found = append(found, fileInterfaces(pass, file)...)
	}

	return found
}

func fileInterfaces(pass *analysis.Pass, file *ast.File) []declaredInterface {
	var found []declaredInterface

	for _, decl := range file.Decls {
		genDecl, isGen := decl.(*ast.GenDecl)
		if !isGen || genDecl.Tok != token.TYPE {
			continue
		}

		found = append(found, specInterfaces(pass, genDecl)...)
	}

	return found
}

func specInterfaces(pass *analysis.Pass, genDecl *ast.GenDecl) []declaredInterface {
	var found []declaredInterface

	for _, spec := range genDecl.Specs {
		typeSpec, isType := spec.(*ast.TypeSpec)
		if !isType {
			continue
		}

		decl, usable := interfaceDecl(pass, typeSpec)
		if usable {
			found = append(found, decl)
		}
	}

	return found
}

// An empty interface constrains nothing and a type-set constraint is not
// implemented by ordinary types, so neither is countable.
func interfaceDecl(pass *analysis.Pass, typeSpec *ast.TypeSpec) (declaredInterface, bool) {
	if _, isIface := typeSpec.Type.(*ast.InterfaceType); !isIface {
		return declaredInterface{}, false
	}

	obj, defined := pass.TypesInfo.Defs[typeSpec.Name].(*types.TypeName)
	if !defined {
		return declaredInterface{}, false
	}

	iface, isIface := obj.Type().Underlying().(*types.Interface)
	if !isIface || iface.NumMethods() == 0 || !iface.IsMethodSet() {
		return declaredInterface{}, false
	}

	return declaredInterface{name: typeSpec.Name.Name, iface: iface, self: obj.Type(), pos: typeSpec.Name.Pos()}, true
}

func namedTypes(pass *analysis.Pass) []*types.TypeName {
	scope := pass.Pkg.Scope()

	out := make([]*types.TypeName, 0, len(scope.Names()))

	for _, name := range scope.Names() {
		typeName, isType := scope.Lookup(name).(*types.TypeName)
		if isType && countableType(typeName) {
			out = append(out, typeName)
		}
	}

	return out
}

// countableType reports whether a named type can stand as an implementation.
// Interfaces are excluded because embedding is not implementing, and generic
// types are excluded because an uninstantiated one has no settled method set.
func countableType(typeName *types.TypeName) bool {
	named, isNamed := typeName.Type().(*types.Named)
	if !isNamed || named.TypeParams().Len() > 0 {
		return false
	}

	_, isIface := named.Underlying().(*types.Interface)

	return !isIface
}

// The message names the scope of the count, so a reader knows the claim
// covers this package and not the whole module.
func reportLoneImpl(pass *analysis.Pass, decl declaredInterface, implementers []*types.TypeName) {
	var (
		count int
		sole  string
	)

	for _, typeName := range implementers {
		if !implementsInterface(typeName, decl) {
			continue
		}

		count++
		sole = typeName.Name()
	}

	if count != 1 {
		return
	}

	pass.Reportf(decl.pos, "[agent] interface %s has exactly one implementing type in this package (%s) and no test double; using %s directly removes the interface and the indirection at its call sites", decl.name, sole, sole)
}

// A type satisfying the interface both as a value and through its pointer is
// still one implementation, so either match counts once.
func implementsInterface(typeName *types.TypeName, decl declaredInterface) bool {
	if types.Identical(typeName.Type(), decl.self) {
		return false
	}

	return implementsByValueOrPointer(typeName.Type(), decl.iface)
}

// Methods declared on *T count, since a caller holding the address of a T
// value can use it wherever the interface is wanted.
func implementsByValueOrPointer(typ types.Type, iface *types.Interface) bool {
	return types.Implements(typ, iface) || types.Implements(types.NewPointer(typ), iface)
}
