// Package analyzers provides custom Go analysis rules for cairnlint.
// Each analyzer is registered via a constructor function in All() and
// uses the golang.org/x/tools/go/analysis framework for AST inspection
// with full type information.
package analyzers
