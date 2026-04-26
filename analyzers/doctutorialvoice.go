package analyzers

import (
	"go/ast"
	"regexp"

	"golang.org/x/tools/go/analysis"
)

// tutorialVoicePattern matches instructional phrases that the Go doc
// comment rules forbid. Comments must use declarative voice that states
// what the entity does, not tutorial narration aimed at the reader.
var tutorialVoicePattern = regexp.MustCompile(
	`(?i)\b(Lets (?:you|tests|callers)|Use this to|You can use this to|Allows you to|Simply call|Just pass in|Here we|Here you)\b`,
)

// docTutorialVoiceAnalyzer returns an analyzer that flags instructional
// voice in doc comments. Reader-addressing phrases (forms of let, use,
// allow, and here) read as tutorial prose rather than API docs; Go
// convention reserves doc comments for declarative voice.
func docTutorialVoiceAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "doctutorialvoice",
		Doc:  "flags tutorial/instructional voice in doc comments; use declarative voice stating what the entity does",
		Run:  runDocTutorialVoice,
	}
}

func runDocTutorialVoice(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			doc := declDoc(decl)
			if doc == nil {
				continue
			}

			for _, comment := range doc.List {
				if tutorialVoicePattern.MatchString(comment.Text) {
					pass.Reportf(comment.Pos(),
						"tutorial voice in doc comment; rewrite in declarative voice describing the entity")
				}
			}
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// declDoc returns the doc comment group attached to decl, or nil if the
// declaration has no doc. Handles both FuncDecl and GenDecl shapes used
// by top-level Go declarations.
func declDoc(decl ast.Decl) *ast.CommentGroup {
	switch typed := decl.(type) {
	case *ast.FuncDecl:
		return typed.Doc
	case *ast.GenDecl:
		return typed.Doc
	}

	return nil
}
