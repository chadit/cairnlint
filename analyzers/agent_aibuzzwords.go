package analyzers

import (
	"go/token"
	"regexp"

	"golang.org/x/tools/go/analysis"
)

// aiBuzzwordPatterns groups the writing-style rule's forbidden vocabulary
// into categories so the diagnostic message points the reader at which
// rule section to revisit. Patterns are case-insensitive and use word
// boundaries on single-word entries to avoid matching substrings.
//
// Each pattern stays ASCII-only on the left-hand side because Go source
// comments routinely contain ASCII text; the right-hand side is the
// human-facing category name used in the diagnostic.
//
//nolint:gochecknoglobals // read-only table shared by every pass
var aiBuzzwordPatterns = []struct {
	category string
	pattern  *regexp.Regexp
}{
	{
		category: "AI buzzword",
		pattern: regexp.MustCompile(
			`(?i)\b(delve|robust|resilient|innovate|innovative|extensive|leverage|utilize|seamless|seamlessly|empower|empowering|comprehensive|holistic|streamline|paradigm|synergy|tapestry|kaleidoscope|realm|pivotal|nuanced|vital|foster|showcase|meticulous|harness|embark|aligns|underscores)\b`,
		),
	},
	{
		category: "AI buzzword phrase",
		pattern: regexp.MustCompile(
			`(?i)(dive deep|dive in|cutting-edge|state-of-the-art|next-generation|game-changing|at the forefront|nail on the head|smoking gun|silver bullet|low hanging fruit|touch base|circle back|move the needle|drive results|take it to the next level|at the end of the day|ever-evolving landscape|best practices|unlock the potential|unleash the power|pave the way)`,
		),
	},
	{
		category: "hedging phrase",
		pattern: regexp.MustCompile(
			`(?i)(generally speaking|it can be argued that|it is worth noting that|it is important to consider|it could be said that|some people say|studies show|experts believe)`,
		),
	},
	{
		category: "formal transition",
		pattern: regexp.MustCompile(
			`(?i)(\b(furthermore|moreover|additionally|subsequently|accordingly|consequently)\b|it is important to note|it should be noted|in conclusion|to summarize)`,
		),
	},
	{
		category: "opening cliche",
		pattern: regexp.MustCompile(
			`(?i)(in today's world|in today's fast-paced world|in the age of|as we all know|it is widely known that|it has been observed that|in the ever-evolving landscape)`,
		),
	},
	{
		category: "academic transition",
		pattern: regexp.MustCompile(
			`(?i)(let's look at a real-world scenario|let's take a look at|imagine you are|picture this|picture yourself|consider the following|at its core|at a fundamental level|when you break it down|when you really think about it)`,
		),
	},
	{
		category: "closing cliche",
		pattern: regexp.MustCompile(
			`(?i)(by following these steps|by internalizing these principles|all in all)`,
		),
	},
	{
		category: "preachy universal",
		pattern: regexp.MustCompile(
			`(?i)\b(we all|everyone knows|as humans|in society)\b`,
		),
	},
}

// agentAIBuzzwordsAnalyzer returns an agent-only analyzer that scans
// comments for AI-flavored vocabulary, hedging, formal transitions,
// clichés, and preachy universals. False-positive risk is real because
// some flagged words (robust, critical, extensive) appear legitimately
// in technical prose. Agent mode lets an LLM triage the hits.
func agentAIBuzzwordsAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "aibuzzwords",
		Doc:  "[agent] flags AI-flavored vocabulary, hedging, and clichés in comments (writing-style rules)",
		Run:  runAgentAIBuzzwords,
	}
}

func runAgentAIBuzzwords(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		for _, cg := range file.Comments {
			for _, comment := range cg.List {
				checkCommentForBuzzwords(pass, comment.Pos(), comment.Text)
			}
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkCommentForBuzzwords reports one diagnostic per category match on
// the comment at pos. Multiple categories can fire on the same comment,
// each surfaced separately so an LLM can triage by rule section.
func checkCommentForBuzzwords(pass *analysis.Pass, pos token.Pos, text string) {
	for _, entry := range aiBuzzwordPatterns {
		match := entry.pattern.FindString(text)
		if match == "" {
			continue
		}

		pass.Reportf(pos, "[agent] %s %q in comment; check writing-style rules", entry.category, match)
	}
}
