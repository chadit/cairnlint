// Package plugin registers cairnlint as a golangci-lint module plugin.
// Import this package from a .custom-gcl.yml configuration to include
// all cairnlint analyzers in a custom golangci-lint build.
//
// The agent-only tier joins the set under the same environment check the
// cairnlint binary makes, [analyzers.DetectAgentCaller], so one environment
// yields one analyzer set from either entry point. The binary's --agent flag
// has no counterpart here; set CAIRNLINT_AGENT=1 instead.
//
// Agent findings do not go to the temp file the binary writes. golangci-lint
// caches each package's issues and skips the analyzers on a cache hit, so a
// file written as a side effect of running them would come up empty on every
// warm run. They are reported through golangci-lint like any other finding,
// prefixed with the analyzer name, and its cache, //nolint handling, and
// output formats apply to them unchanged.
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
	set := analyzers.All()
	if analyzers.DetectAgentCaller() {
		set = append(set, analyzers.AgentOnly()...)
	}

	return set, nil
}

//nolint:revive,nogetterprefix // receiver unused and Get prefix both fixed by golangci-lint's LinterPlugin interface
func (c *cairnlintPlugin) GetLoadMode() string {
	// cairnlint uses type information for call resolution,
	// struct field type checking, and scope analysis.
	return register.LoadModeTypesInfo
}
