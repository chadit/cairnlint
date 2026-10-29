package plugin_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/chadit/cairnlint/analyzers"
	_ "github.com/chadit/cairnlint/plugin"
)

// clearEnvironment blanks every variable for the rest of the test. The tests
// themselves often run under an agent, and DetectAgentCaller checks a list
// that grows, so blanking all of them is the only way to get a human caller.
func clearEnvironment(t *testing.T) {
	t.Helper()

	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		t.Setenv(name, "")
	}
}

// pluginAnalyzerNames builds the analyzers the registered plugin hands to
// golangci-lint and returns their names.
func pluginAnalyzerNames(t *testing.T) []string {
	t.Helper()

	newPlugin, err := register.GetPlugin("cairnlint")
	if err != nil {
		t.Fatalf("get registered plugin: %v", err)
	}

	linterPlugin, err := newPlugin(nil)
	if err != nil {
		t.Fatalf("construct plugin: %v", err)
	}

	built, err := linterPlugin.BuildAnalyzers()
	if err != nil {
		t.Fatalf("build analyzers: %v", err)
	}

	return analyzerNames(built)
}

func analyzerNames(set []*analysis.Analyzer) []string {
	names := make([]string, len(set))
	for idx, analyzer := range set {
		names[idx] = analyzer.Name
	}

	return names
}

// TestPluginAnalyzerSetFollowsAgentDetection verifies the plugin builds the
// same set the cairnlint binary runs for each environment: the standard tier
// for a human caller, and the agent tier on top when an agent is detected.
func TestPluginAnalyzerSetFollowsAgentDetection(t *testing.T) {
	standard := analyzerNames(analyzers.All())

	agentTier := analyzerNames(analyzers.AgentOnly())
	if len(agentTier) == 0 {
		t.Fatal("AgentOnly is empty, so the two environments cannot differ")
	}

	cases := []struct {
		name     string
		agentVar string
		want     []string
	}{
		{name: "human caller", want: standard},
		{name: "explicit opt-in", agentVar: "CAIRNLINT_AGENT", want: slices.Concat(standard, agentTier)},
		{name: "Claude Code", agentVar: "CLAUDECODE", want: slices.Concat(standard, agentTier)},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			clearEnvironment(t)

			if testCase.agentVar != "" {
				t.Setenv(testCase.agentVar, "1")
			}

			if got := pluginAnalyzerNames(t); !slices.Equal(got, testCase.want) {
				t.Errorf("plugin analyzers = %v\nwant %v", got, testCase.want)
			}
		})
	}
}
