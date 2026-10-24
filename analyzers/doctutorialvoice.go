package analyzers

import (
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

// tutorialVoicePattern matches instructional phrases that the Go doc
// comment rules forbid. Comments must use declarative voice that states
// what the entity does, not tutorial narration aimed at the reader.
var tutorialVoicePattern = regexp.MustCompile(
	`(?i)\b(Lets (?:you|tests|callers)|Use this to|You can use this to|Allows you to|Simply call|Just pass in|Here we|Here you)\b`,
)

// imperativeOpenerPattern matches a sentence opening with a reader-directed
// verb followed by a space and a word. Requiring the second word skips
// "Run," "Run-time", and "Run." where the word names something instead of
// instructing. "See" is left out because "See RFC 7230" is a cross-reference.
var imperativeOpenerPattern = regexp.MustCompile("^(Run|Use|Call|Invoke|Execute) +(\\[|`|\\pL+)")

// sentenceBreakPattern finds where one sentence ends and the next begins
// inside a prose line.
var sentenceBreakPattern = regexp.MustCompile(`[.!?]\s+`)

// listMarkerPattern matches an indented list item marker, using the bullets
// and number forms go/doc/comment accepts, which it reads as a list rather
// than a code block.
var listMarkerPattern = regexp.MustCompile(`^[ \t]+(?:[-*+•]|\d+[.)])(?:[ \t]+|$)`)

// goTestPattern matches a go test command anywhere in a code block line,
// because run instructions often wrap it in docker or make invocations.
var goTestPattern = regexp.MustCompile(`\bgo test\b`)

// commentMarkerLen is the byte length of the "//" that starts a line
// comment, needed to turn an offset in the stripped text into a position.
const commentMarkerLen = len("//")

// minSpaceIndent is the space-only indent that marks a code block line:
// go/doc/comment strips the one space every line shares, so code needs two.
const minSpaceIndent = 2

// docLineKind classifies a doc comment line by the block it belongs to.
type docLineKind int

const (
	proseLine docLineKind = iota
	blankLine
	codeLine
	listItemLine
	listTextLine
)

// docLine is one "//" doc comment line with the marker stripped. blockStart
// marks the first line of a prose paragraph, where a sentence starts.
type docLine struct {
	comment    *ast.Comment
	text       string
	kind       docLineKind
	blockStart bool
}

// docTutorialVoiceAnalyzer returns an analyzer that flags instructional
// voice in doc comments. Reader-addressing phrases, imperative sentence
// openers, and go test run instructions read as tutorial prose rather than
// API docs; Go convention reserves doc comments for declarative voice.
func docTutorialVoiceAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "doctutorialvoice",
		Doc: "flags instructional voice in doc comments: tutorial phrases (Lets you, Use this to, Here we) in any doc, " +
			"sentences opening with Run/Use/Call/Invoke/Execute in function docs, " +
			"and go test command lines in test function doc code blocks; use declarative voice stating what the entity does",
		Run: runDocTutorialVoice,
	}
}

func runDocTutorialVoice(pass *analysis.Pass) (any, error) {
	idents := packageIdents(pass.Pkg)

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			checkDeclDoc(pass, decl, idents)
		}
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkDeclDoc applies the phrase list to every doc and the opener and
// go test checks to function docs only, since a type or var doc rarely
// reads as an instruction and the openers there are mostly nouns.
func checkDeclDoc(pass *analysis.Pass, decl ast.Decl, idents map[string]bool) {
	doc := declDoc(decl)
	if doc == nil {
		return
	}

	reportTutorialPhrases(pass, doc)

	funcDecl, ok := decl.(*ast.FuncDecl)
	if !ok {
		return
	}

	lines := docLines(doc)
	reportImperativeOpeners(pass, lines, idents)

	if isTestFile(pass, funcDecl) && isTestRunnerFunc(funcDecl.Name.Name) {
		reportGoTestLines(pass, lines)
	}
}

func reportTutorialPhrases(pass *analysis.Pass, doc *ast.CommentGroup) {
	for _, comment := range doc.List {
		if tutorialVoicePattern.MatchString(comment.Text) {
			pass.Reportf(comment.Pos(),
				"tutorial voice in doc comment; rewrite in declarative voice describing the entity")
		}
	}
}

// docLines returns the "//" lines of doc, each classified. Block comments
// are skipped because their lines carry no per-line position.
func docLines(doc *ast.CommentGroup) []docLine {
	lines := make([]docLine, 0, len(doc.List))
	prevKind := blankLine

	var inList bool

	for _, comment := range doc.List {
		text, ok := strings.CutPrefix(comment.Text, "//")
		if !ok {
			continue
		}

		kind := classifyDocLine(text, inList)
		lines = append(lines, docLine{
			comment:    comment,
			text:       text,
			kind:       kind,
			blockStart: kind == proseLine && prevKind != proseLine,
		})
		prevKind = kind
		inList = kind == listItemLine || kind == listTextLine || (kind == blankLine && inList)
	}

	return lines
}

// classifyDocLine applies go/doc/comment's Go 1.19 rules. An indented line
// is a list item when marked, list text while a list is open, and code
// otherwise, even right after prose. A blank line keeps a list open, so an
// indented line after it continues the item. Indent means a tab or two
// characters, because go/doc/comment strips one common space.
func classifyDocLine(text string, inList bool) docLineKind {
	if strings.TrimSpace(text) == "" {
		return blankLine
	}

	indent := text[:indentWidth(text)]
	if !strings.Contains(indent, "\t") && len(indent) < minSpaceIndent {
		return proseLine
	}

	if listMarkerPattern.MatchString(text) {
		return listItemLine
	}

	if inList {
		return listTextLine
	}

	return codeLine
}

// sentenceScan carries sentence and Deprecated state across the lines of
// one prose paragraph.
type sentenceScan struct {
	deprecated    bool
	sentenceStart bool
}

// starts returns the byte offsets in line.text where a sentence begins.
// List lines return none: a step list usually describes what the function
// does, and nothing here knows who performs the verb. Deprecated
// paragraphs return none, because "Use NewClient instead." is the
// migration note go.dev/doc/comment#deprecations asks for.
func (scan *sentenceScan) starts(line docLine) []int {
	if line.kind != proseLine {
		return nil
	}

	lead := indentWidth(line.text)
	if line.blockStart {
		scan.deprecated = strings.HasPrefix(line.text[lead:], "Deprecated: ")
		scan.sentenceStart = true
	}

	var offsets []int
	if scan.sentenceStart {
		offsets = append(offsets, lead)
	}

	for _, match := range sentenceBreakPattern.FindAllStringIndex(line.text, -1) {
		offsets = append(offsets, match[1])
	}

	scan.sentenceStart = endsSentence(line.text)

	if scan.deprecated {
		return nil
	}

	return offsets
}

// indentWidth returns the byte width of the whitespace run that opens text.
func indentWidth(text string) int {
	return len(text) - len(strings.TrimLeft(text, " \t"))
}

func endsSentence(text string) bool {
	trimmed := strings.TrimRight(text, " ")

	return strings.HasSuffix(trimmed, ".") || strings.HasSuffix(trimmed, "!") || strings.HasSuffix(trimmed, "?")
}

func reportImperativeOpeners(pass *analysis.Pass, lines []docLine, idents map[string]bool) {
	var scan sentenceScan

	for _, line := range lines {
		for _, offset := range scan.starts(line) {
			reportOpenerAt(pass, line, offset, idents)
		}
	}
}

// reportOpenerAt reports an imperative opener at offset unless the word is
// a package identifier, which covers the function's own name, or the
// phrase is a noun compound such as "Use of" or "Call sites".
func reportOpenerAt(pass *analysis.Pass, line docLine, offset int, idents map[string]bool) {
	match := imperativeOpenerPattern.FindStringSubmatch(line.text[offset:])
	if match == nil {
		return
	}

	opener, nextWord := match[1], match[2]
	if isNounCompoundWord(nextWord) || idents[opener] {
		return
	}

	pass.Reportf(line.comment.Pos()+token.Pos(commentMarkerLen+offset),
		"instructional voice in doc comment: a sentence that opens with %q tells the reader what to do; "+
			"rewrite in declarative voice describing the function", opener)
}

// isNounCompoundWord reports whether word, following an opener, makes a
// noun compound spelled like the verb, as in "Call sites" or "Run time", so
// the sentence describes something rather than instructing the reader.
func isNounCompoundWord(word string) bool {
	switch word {
	case "of", "site", "sites", "case", "cases", "time", "times", "count", "counts", "graph", "stack":
		return true
	}

	return false
}

func reportGoTestLines(pass *analysis.Pass, lines []docLine) {
	for _, line := range lines {
		if line.kind == codeLine && goTestPattern.MatchString(line.text) {
			pass.Reportf(line.comment.Pos(),
				"run instruction in test doc comment: a \"go test\" command tells the reader how to run this test; "+
					"drop it or move a concise run note to the package doc of a gated test package")
		}
	}
}

// isTestRunnerFunc applies cmd/go's rule for test, benchmark, and fuzz
// names: a bare prefix such as Test runs, and a lowercase letter after the
// prefix, as in Testdata, marks a helper. Example functions are left out
// because their docs describe printed output, not how to run them.
func isTestRunnerFunc(name string) bool {
	return slices.ContainsFunc([]string{testPrefix, benchmarkPrefix, fuzzPrefix}, func(prefix string) bool {
		suffix, found := strings.CutPrefix(name, prefix)
		if !found {
			return false
		}

		first, _ := utf8.DecodeRuneInString(suffix)

		return !unicode.IsLower(first)
	})
}

// packageIdents collects every name a doc sentence might refer to. An
// external test package also takes the names of the package it tests,
// because black-box test docs mostly name that package's API.
func packageIdents(pkg *types.Package) map[string]bool {
	idents := make(map[string]bool, pkg.Scope().Len())
	addScopeIdents(idents, pkg.Scope())

	if tested := testedPackage(pkg); tested != nil {
		addScopeIdents(idents, tested.Scope())
	}

	return idents
}

// testedPackage returns the package an external test package imports under
// its own path without the _test suffix, or nil for any other package.
func testedPackage(pkg *types.Package) *types.Package {
	if !strings.HasSuffix(pkg.Name(), "_test") {
		return nil
	}

	testedPath := strings.TrimSuffix(pkg.Path(), "_test")
	index := slices.IndexFunc(pkg.Imports(), func(imported *types.Package) bool {
		return imported.Path() == testedPath
	})

	if index < 0 {
		return nil
	}

	return pkg.Imports()[index]
}

// addScopeIdents adds scope names plus the methods and struct fields of
// each named type. Every method in a package belongs to a package-level
// named type, so the method sets cover all declared methods.
func addScopeIdents(idents map[string]bool, scope *types.Scope) {
	for _, name := range scope.Names() {
		idents[name] = true
		addTypeMembers(idents, scope.Lookup(name))
	}
}

// addTypeMembers adds the method set, promoted and interface methods
// included, and the direct struct fields of a named type.
func addTypeMembers(idents map[string]bool, obj types.Object) {
	typeName, isTypeName := obj.(*types.TypeName)
	if !isTypeName {
		return
	}

	named, isNamed := types.Unalias(typeName.Type()).(*types.Named)
	if !isNamed {
		return
	}

	for _, selection := range typeutil.IntuitiveMethodSet(named, nil) {
		idents[selection.Obj().Name()] = true
	}

	structType, isStruct := named.Underlying().(*types.Struct)
	if !isStruct {
		return
	}

	for field := range structType.Fields() {
		idents[field.Name()] = true
	}
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
