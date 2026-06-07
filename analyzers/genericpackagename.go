package analyzers

import (
	"strings"

	"golang.org/x/tools/go/analysis"
)

// genericPackageNames holds package names the style guide calls out as too
// vague. They carry no meaning, so callers tend to rename them on import.
var genericPackageNames = map[string]bool{ //nolint:gochecknoglobals // package-internal lookup table, not mutable state
	"util":      true,
	"utils":     true,
	"utility":   true,
	"utilities": true,
	"common":    true,
	"commons":   true,
	"helper":    true,
	"helpers":   true,
	"misc":      true,
	"shared":    true,
}

// genericPackageNameAnalyzer returns an analyzer that flags packages named with
// a vague catch-all word like util, common, or helper. A package should be
// named for what it provides so its exported names read well at the call site.
func genericPackageNameAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "genericpackagename",
		Doc:  "flags vague package names like util, common, or helper; name the package for what it provides",
		Run:  runGenericPackageName,
	}
}

func runGenericPackageName(pass *analysis.Pass) (any, error) {
	if len(pass.Files) == 0 {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	clause := pass.Files[0].Name
	name := clause.Name

	if name == mainPkgName || strings.HasSuffix(name, "_test") {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	if genericPackageNames[name] {
		pass.Reportf(clause.Pos(), "package name %q is too vague; name the package for what it provides", name)
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}
