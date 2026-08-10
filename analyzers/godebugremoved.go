package analyzers

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// goDebugDirective is the comment prefix that pins a GODEBUG setting for the
// main package that contains it.
const goDebugDirective = "//go:debug"

// Releases in which the settings below stopped being the default. Named so the
// table reads without repeating the literal.
const (
	goDebugSince122 = "Go 1.22"
	goDebugSince123 = "Go 1.23"
)

// removedGoDebugSettings maps each GODEBUG setting deleted in Go 1.27 to the
// release whose behavior it was holding back. Naming that release tells the
// reader how long the old behavior has been on borrowed time.
var removedGoDebugSettings = map[string]string{ //nolint:gochecknoglobals // package-internal lookup table
	"asynctimerchan":  goDebugSince123,
	"gotypesalias":    goDebugSince123,
	"tls10server":     goDebugSince122,
	"tls3des":         goDebugSince123,
	"tlsrsakex":       goDebugSince122,
	"tlsunsafeekm":    goDebugSince122,
	"x509keypairleaf": goDebugSince123,
}

// goDebugRemovedAnalyzer returns an analyzer that flags //go:debug directives
// naming a GODEBUG setting Go 1.27 deleted.
//
// The go command only tolerates a deleted setting at its final default value,
// so a directive still pinning the old behavior stops the build rather than
// degrading quietly. Catching it here beats finding it during a toolchain
// upgrade.
func goDebugRemovedAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "godebugremoved",
		Doc:  "flags //go:debug directives naming a GODEBUG setting removed in Go 1.27",
		Run:  runGoDebugRemoved,
	}
}

func runGoDebugRemoved(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		reportRemovedGoDebug(pass, file)
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// reportRemovedGoDebug flags every deleted GODEBUG setting pinned by file.
//
// Diagnostics land on the package clause rather than the directive itself. A
// //go:debug value may not contain a space, so the directive line has no room
// for a suppression or expectation comment, and the setting name in the message
// identifies which line above needs the edit.
func reportRemovedGoDebug(pass *analysis.Pass, file *ast.File) {
	for _, setting := range pinnedGoDebugSettings(file) {
		since, wasRemoved := removedGoDebugSettings[setting]
		if !wasRemoved {
			continue
		}

		if !goVersionAtLeast(pass, file.Name.Pos(), goVersion127) {
			continue
		}

		pass.Reportf(file.Name.Pos(),
			"GODEBUG setting %s was removed in Go 1.27; it stopped being the default in %s and the go command now rejects any other value",
			setting, since)
	}
}

// pinnedGoDebugSettings returns the setting names that file pins with a
// //go:debug directive.
//
// Only comments above the package clause count. The go command honors the
// directive nowhere else, so flagging the same text from inside a function
// would report a line that has no effect on the build.
func pinnedGoDebugSettings(file *ast.File) []string {
	var settings []string

	for _, group := range file.Comments {
		if group.End() > file.Package {
			break
		}

		for _, comment := range group.List {
			if setting, isDirective := goDebugSetting(comment.Text); isDirective {
				settings = append(settings, setting)
			}
		}
	}

	return settings
}

// goDebugSetting returns the setting name from a //go:debug directive comment.
func goDebugSetting(text string) (string, bool) {
	rest, isDirective := strings.CutPrefix(text, goDebugDirective)
	if !isDirective {
		return "", false
	}

	setting, _, hasValue := strings.Cut(strings.TrimSpace(rest), "=")
	if !hasValue {
		return "", false
	}

	return strings.TrimSpace(setting), true
}
