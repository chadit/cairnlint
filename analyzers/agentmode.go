package analyzers

import (
	"os"

	"golang.org/x/tools/go/analysis"
)

// DetectAgentCaller checks environment variables set by popular AI coding
// tools to determine if cairnlint is being invoked by an LLM agent rather
// than a human. When true, callers should enable the AgentOnly analyzers
// which produce heuristic diagnostics suitable for LLM triage.
func DetectAgentCaller() bool {
	checks := []string{
		// Emerging standards (Vercel convention / agents.md convention)
		"AI_AGENT",
		"AGENT",

		// Claude Code (Anthropic) — confirmed in official docs
		"CLAUDECODE",

		// OpenAI Codex CLI — confirmed in source (spawn.rs, exec_env.rs)
		"CODEX_SANDBOX",
		"CODEX_THREAD_ID",

		// Gemini CLI (Google) — confirmed in official docs
		"GEMINI_CLI",

		// Cursor — confirmed (re-added after bug, forum.cursor.com)
		"CURSOR_AGENT",

		// Qwen Code (Alibaba) — confirmed in official docs
		"QWEN_CODE",

		// Goose (Block) — confirmed in GitHub PR #3911
		"GOOSE_TERMINAL",

		// Cline — confirmed in agents.md#136 discussion
		"CLINE_ACTIVE",

		// Augment Code (Auggie) — confirmed in agents.md#136 discussion
		"AUGMENT_AGENT",

		// TRAE AI — confirmed in agents.md#136 discussion
		"TRAE_AI_SHELL_ID",

		// OpenCode (sst/opencode) — suspected from source code
		"OPENCODE_CLIENT",

		// Explicit opt-in for tools that don't set identifying vars
		// (Aider, Continue.dev, Windsurf, Amazon Q — confirmed absent)
		"CAIRNLINT_AGENT",
	}

	for _, key := range checks {
		if os.Getenv(key) != "" {
			return true
		}
	}

	return false
}

// AgentOnly returns analyzers that are only enabled in agent mode. These
// produce heuristic-based diagnostics with a higher false-positive rate
// than the standard set. LLM agents can triage them; humans would find
// the noise annoying.
//
// Diagnostic messages from these analyzers are prefixed with [agent] so
// they're easy to filter and identify in output.
func AgentOnly() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		agentExportedInTestFileAnalyzer(),
	}
}
