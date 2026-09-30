package require_os_eol

import (
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func buildHardcodedNewlineMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "hardcodedNewline",
		Description: `Hardcoded "\n" is not portable.`,
		Help:        `Import { EOL } from "node:os" and use EOL instead.`,
	}
}

// Parents where replacing the literal with a template literal is a syntax
// error, so the violation is reported but not fixed.
var unfixableParents = []ast.Kind{
	ast.KindEnumMember,
	ast.KindExportDeclaration,
	ast.KindImportAttribute,
	ast.KindImportDeclaration,
	ast.KindJsxAttribute,
	ast.KindLiteralType,
	ast.KindModuleDeclaration,
	ast.KindTaggedTemplateExpression,
}

type escape struct {
	length int
	start  int
}

// Every unescaped `\n` / `\r\n` escape in a literal's raw source text. Counting
// the backslashes by hand is what distinguishes `"a\\nb"` (a literal backslash
// followed by an n) from `"a\nb"` (a newline).
func newlineEscapes(raw string) []escape {
	found := []escape{}
	for i := 0; i < len(raw); {
		if raw[i] != '\\' || i+1 >= len(raw) {
			i++
			continue
		}
		switch {
		case raw[i+1] == 'n':
			found = append(found, escape{length: 2, start: i})
			i += 2
		case raw[i+1] == 'r' && i+3 < len(raw) && raw[i+2] == '\\' && raw[i+3] == 'n':
			found = append(found, escape{length: 4, start: i})
			i += 4
		default:
			i += 2
		}
	}
	return found
}

func replaceEscapes(raw string, placeholder string) string {
	var built strings.Builder
	previous := 0
	for _, found := range newlineEscapes(raw) {
		built.WriteString(raw[previous:found.start])
		built.WriteString(placeholder)
		previous = found.start + found.length
	}
	built.WriteString(raw[previous:])
	return built.String()
}

func escapeForTemplate(raw string) string {
	return strings.ReplaceAll(strings.ReplaceAll(raw, "`", "\\`"), "${", "\\${")
}

var RequireOsEolRule = rule.Rule{
	Name: "require-os-eol",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		text := ctx.SourceFile.Text()

		eolLocalName := "EOL"
		eolAlreadyImported := false
		importFixEmitted := false
		var osImport *ast.Node
		var lastImport *ast.Node
		var firstStatement *ast.Node

		sourceOf := func(node *ast.Node) string {
			trimmed := utils.TrimNodeTextRange(ctx.SourceFile, node)
			return text[trimmed.Pos():trimmed.End()]
		}

		// The span of a template part's literal text, with the delimiters the
		// scanner swallowed (`` ` ``, `${`, `}`) stripped back off. It starts from
		// the trimmed token position, not node.Pos(), which points at the leading
		// trivia and would keep the opening backtick in the chunk.
		chunkRange := func(node *ast.Node) core.TextRange {
			start := utils.TrimNodeTextRange(ctx.SourceFile, node).Pos() + 1
			end := node.End() - 1
			if node.Kind == ast.KindTemplateHead || node.Kind == ast.KindTemplateMiddle {
				end = node.End() - 2
			}
			if start > end || start < 0 || end > len(text) {
				return core.NewTextRange(0, 0)
			}
			return core.NewTextRange(start, end)
		}

		rawChunk := func(node *ast.Node) string {
			span := chunkRange(node)
			return text[span.Pos():span.End()]
		}

		// One import fix per file, whatever shape the existing `node:os` import has.
		importFix := func() []rule.RuleFix {
			if eolAlreadyImported || importFixEmitted {
				return nil
			}
			importFixEmitted = true
			if osImport != nil {
				clause := osImport.AsImportDeclaration().ImportClause
				if clause != nil {
					bindings := clause.AsImportClause().NamedBindings
					if bindings != nil && bindings.Kind == ast.KindNamedImports {
						elements := bindings.AsNamedImports().Elements.Nodes
						if len(elements) > 0 {
							return []rule.RuleFix{rule.RuleFixInsertBefore(ctx.SourceFile, elements[0], "EOL, ")}
						}
					}
					if name := clause.Name(); name != nil {
						return []rule.RuleFix{rule.RuleFixInsertAfter(name, ", { EOL }")}
					}
				}
			}
			if lastImport != nil {
				return []rule.RuleFix{rule.RuleFixInsertAfter(lastImport, "\nimport { EOL } from \"node:os\";")}
			}
			if firstStatement != nil {
				return []rule.RuleFix{rule.RuleFixInsertBefore(ctx.SourceFile, firstStatement, "import { EOL } from \"node:os\";\n")}
			}
			return nil
		}

		isFixable := func(node *ast.Node) bool {
			parent := node.Parent
			if parent == nil || slices.Contains(unfixableParents, parent.Kind) {
				return false
			}
			// A member's name is not an expression position.
			return parent.Name() != node
		}

		return rule.RuleListeners{
			ast.KindSourceFile: func(node *ast.Node) {
				statements := node.AsSourceFile().Statements.Nodes
				if len(statements) > 0 {
					firstStatement = statements[0]
				}
				for _, statement := range statements {
					if statement.Kind != ast.KindImportDeclaration {
						continue
					}
					lastImport = statement
					declaration := statement.AsImportDeclaration()
					specifier := declaration.ModuleSpecifier
					if !ast.IsStringLiteral(specifier) ||
						(specifier.Text() != "node:os" && specifier.Text() != "os") {
						continue
					}
					if osImport == nil {
						osImport = statement
					}
					clause := declaration.ImportClause
					if clause == nil {
						continue
					}
					bindings := clause.AsImportClause().NamedBindings
					if bindings == nil || bindings.Kind != ast.KindNamedImports {
						continue
					}
					for _, element := range bindings.AsNamedImports().Elements.Nodes {
						specifier := element.AsImportSpecifier()
						imported := specifier.PropertyName
						if imported == nil {
							imported = specifier.Name()
						}
						if imported != nil && imported.Text() == "EOL" {
							eolAlreadyImported = true
							if local := specifier.Name(); local != nil {
								eolLocalName = local.Text()
							}
						}
					}
				}
			},

			ast.KindStringLiteral: func(node *ast.Node) {
				raw := sourceOf(node)
				if len(newlineEscapes(raw)) == 0 {
					return
				}
				if !isFixable(node) {
					ctx.ReportNode(node, buildHardcodedNewlineMessage())
					return
				}
				body := replaceEscapes(escapeForTemplate(raw[1:len(raw)-1]), "${"+eolLocalName+"}")
				replacement := "`" + body + "`"
				if body == "${"+eolLocalName+"}" {
					replacement = eolLocalName
				}
				ctx.ReportNodeWithFixes(node, buildHardcodedNewlineMessage(), func() []rule.RuleFix {
					return append([]rule.RuleFix{rule.RuleFixReplace(ctx.SourceFile, node, replacement)}, importFix()...)
				})
			},

			ast.KindNoSubstitutionTemplateLiteral: func(node *ast.Node) {
				if node.Parent != nil && node.Parent.Kind == ast.KindTaggedTemplateExpression {
					return
				}
				chunk := rawChunk(node)
				if len(newlineEscapes(chunk)) == 0 {
					return
				}
				if !isFixable(node) {
					ctx.ReportNode(node, buildHardcodedNewlineMessage())
					return
				}
				ctx.ReportNodeWithFixes(node, buildHardcodedNewlineMessage(), func() []rule.RuleFix {
					replaced := replaceEscapes(chunk, "${"+eolLocalName+"}")
					return append(
						[]rule.RuleFix{rule.RuleFixReplaceRange(chunkRange(node), replaced)},
						importFix()...,
					)
				})
			},

			ast.KindTemplateExpression: func(node *ast.Node) {
				if node.Parent != nil && node.Parent.Kind == ast.KindTaggedTemplateExpression {
					return
				}
				expression := node.AsTemplateExpression()
				parts := []*ast.Node{expression.Head}
				for _, span := range expression.TemplateSpans.Nodes {
					parts = append(parts, span.AsTemplateSpan().Literal)
				}

				dirty := []*ast.Node{}
				for _, part := range parts {
					if len(newlineEscapes(rawChunk(part))) > 0 {
						dirty = append(dirty, part)
					}
				}
				if len(dirty) == 0 {
					return
				}
				if !isFixable(node) {
					ctx.ReportNode(node, buildHardcodedNewlineMessage())
					return
				}
				ctx.ReportNodeWithFixes(node, buildHardcodedNewlineMessage(), func() []rule.RuleFix {
					fixes := []rule.RuleFix{}
					for _, part := range dirty {
						replaced := replaceEscapes(rawChunk(part), "${"+eolLocalName+"}")
						fixes = append(fixes, rule.RuleFixReplaceRange(chunkRange(part), replaced))
					}
					return append(fixes, importFix()...)
				})
			},
		}
	},
}
