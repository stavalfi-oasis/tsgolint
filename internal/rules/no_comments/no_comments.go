package no_comments

import (
	"regexp"
	"slices"
	"sort"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

// Comments the tools need in order to work: lint control, type-checker
// directives, triple-slash references, and the shebang.
var toolDirective = regexp.MustCompile(`^\s*(?://|/\*)?\s*(?:oxlint|eslint|prettier|oxfmt|@ts-|/\s*<reference|#!)`)

var commentRangeNodeFactory = ast.NewNodeFactory(ast.NodeFactoryHooks{})

func buildNoCommentsMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "comment",
		Description: "Comments are banned.",
		Help:        "Delete it, or put what it says into a name the code already reads.",
	}
}

// Every comment in the file, in source order. A comment is trivia, so it is
// only reachable from the position of the token it precedes or follows; walking
// every node and asking at each boundary is what makes the sweep complete.
func allComments(sourceFile *ast.SourceFile) []core.TextRange {
	text := sourceFile.Text()
	seen := map[int]bool{}
	ranges := []core.TextRange{}

	collect := func(pos int) {
		for commentRange := range scanner.GetLeadingCommentRanges(commentRangeNodeFactory, text, pos) {
			if !seen[commentRange.Pos()] {
				seen[commentRange.Pos()] = true
				ranges = append(ranges, core.NewTextRange(commentRange.Pos(), commentRange.End()))
			}
		}
		for commentRange := range scanner.GetTrailingCommentRanges(commentRangeNodeFactory, text, pos) {
			if !seen[commentRange.Pos()] {
				seen[commentRange.Pos()] = true
				ranges = append(ranges, core.NewTextRange(commentRange.Pos(), commentRange.End()))
			}
		}
	}

	var walk func(node *ast.Node) bool
	walk = func(node *ast.Node) bool {
		collect(node.Pos())
		collect(node.End())
		ast.ForEachChildAndJSDoc(node, sourceFile, walk)
		return false
	}
	walk(sourceFile.AsNode())

	sort.Slice(ranges, func(i int, j int) bool { return ranges[i].Pos() < ranges[j].Pos() })
	return ranges
}

// Widen a comment to the whitespace around it, and to its trailing newline when
// the comment is the only thing on its line — otherwise deleting it leaves a
// blank line behind.
func wholeLineRange(text string, commentRange core.TextRange) core.TextRange {
	start := commentRange.Pos()
	end := commentRange.End()
	for start > 0 && (text[start-1] == ' ' || text[start-1] == '\t') {
		start--
	}
	for end < len(text) && (text[end] == ' ' || text[end] == '\t') {
		end++
	}
	ownsTheLine := start == 0 || text[start-1] == '\n'
	if ownsTheLine && end < len(text) && text[end] == '\n' {
		end++
	}
	return core.NewTextRange(start, end)
}

var NoCommentsRule = rule.Rule{
	Name: "no-comments",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindSourceFile: func(node *ast.Node) {
				text := ctx.SourceFile.Text()
				runs := []core.TextRange{}

				for _, commentRange := range allComments(ctx.SourceFile) {
					if toolDirective.MatchString(text[commentRange.Pos():commentRange.End()]) {
						continue
					}
					widened := wholeLineRange(text, commentRange)
					// Consecutive comment lines are one run, so a block of them
					// reports and deletes as a single unit.
					if last := len(runs) - 1; last >= 0 && runs[last].End() >= widened.Pos() {
						runs[last] = runs[last].WithEnd(max(runs[last].End(), widened.End()))
						continue
					}
					runs = append(runs, widened)
				}

				for _, run := range slices.Clone(runs) {
					reported := run
					ctx.ReportDiagnosticWithFixes(rule.RuleDiagnostic{
						Range:      reported,
						RuleName:   "no-comments",
						Message:    buildNoCommentsMessage(),
						SourceFile: ctx.SourceFile,
					}, func() []rule.RuleFix {
						return []rule.RuleFix{rule.RuleFixRemoveRange(reported)}
					})
				}
			},
		}
	},
}
