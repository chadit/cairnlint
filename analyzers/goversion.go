package analyzers

import (
	"go/ast"
	"go/token"
	"go/version"

	"golang.org/x/tools/go/analysis"
)

// Go language versions that gate an analyzer's suggestion. Named constants
// keep the release that introduced an API visible at the call site and stop
// goconst from flipping once several analyzers reference the same release.
const (
	goVersion118 = "go1.18"
	goVersion119 = "go1.19"
	goVersion124 = "go1.24"
	goVersion125 = "go1.25"
	goVersion126 = "go1.26"
	goVersion127 = "go1.27"
)

// goVersionAtLeast reports whether the Go language version in force for the
// file containing pos is want or newer.
//
// Analyzers that recommend a standard library API must gate on this. Naming an
// API the caller's language version does not have is worse than staying quiet:
// Go 1.27 runs the stdversion vet check as part of `go test` by default, so
// acting on an ungated suggestion converts a lint hint into a vet failure on
// every subsequent test run.
func goVersionAtLeast(pass *analysis.Pass, pos token.Pos, want string) bool {
	return versionAtLeast(effectiveGoVersion(pass, pos), want)
}

// versionAtLeast compares two Go version strings of the form "go1.27".
//
// An unparseable or empty have reports true so that packages loaded without
// module information keep their diagnostics. Silently dropping every
// version-dependent rule would be a far quieter failure than the occasional
// suggestion aimed at a version we could not confirm.
func versionAtLeast(have, want string) bool {
	if !version.IsValid(have) {
		return true
	}

	return version.Compare(have, want) >= 0
}

// effectiveGoVersion returns the language version governing the file that
// contains pos, falling back to the package's version.
//
// The per-file map is consulted first because a //go:build go1.N constraint
// lowers the language version for that one file below the go directive in
// go.mod, and it is the file's version that the compiler and stdversion apply.
func effectiveGoVersion(pass *analysis.Pass, pos token.Pos) string {
	if pass.TypesInfo != nil {
		if file := fileContaining(pass, pos); file != nil {
			if fileVersion := pass.TypesInfo.FileVersions[file]; fileVersion != "" {
				return fileVersion
			}
		}
	}

	if pass.Pkg == nil {
		return ""
	}

	return pass.Pkg.GoVersion()
}

// fileContaining returns the parsed file whose byte range covers pos, or nil
// when pos belongs to a file outside the pass.
func fileContaining(pass *analysis.Pass, pos token.Pos) *ast.File {
	for _, file := range pass.Files {
		if file.FileStart <= pos && pos < file.FileEnd {
			return file
		}
	}

	return nil
}
