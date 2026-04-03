// Package plugin registers cairnlint as a golangci-lint module plugin.
// Import this package from a .custom-gcl.yml configuration to include
// all cairnlint analyzers in a custom golangci-lint build.
package plugin

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/chadit/cairnlint/analyzers"
)

//nolint:gochecknoinits // required by golangci-lint plugin registration API
func init() {
	register.Plugin("cairnlint", newPlugin)
}

type cairnlintPlugin struct{}

func newPlugin(_ any) (register.LinterPlugin, error) {
	return &cairnlintPlugin{}, nil
}

//nolint:revive // receiver unused; interface contract from golangci-lint plugin API
func (c *cairnlintPlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return analyzers.All(), nil
}

//nolint:revive // receiver unused; interface contract from golangci-lint plugin API
func (c *cairnlintPlugin) GetLoadMode() string {
	// cairnlint uses type information for call resolution,
	// struct field type checking, and scope analysis.
	return register.LoadModeTypesInfo
}
