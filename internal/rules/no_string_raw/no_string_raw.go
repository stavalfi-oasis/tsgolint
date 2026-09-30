package no_string_raw

import (
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func buildStringRawMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "stringRaw",
		Description: "String.raw is banned.",
		Help:        "Write a plain template literal and double the backslashes.",
	}
}

func isStringRawTag(tag *ast.Node) bool {
	if !ast.IsPropertyAccessExpression(tag) {
		return false
	}
	access := tag.AsPropertyAccessExpression()
	return ast.IsIdentifier(access.Expression) &&
		access.Expression.Text() == "String" &&
		access.Name().Text() == "raw"
}

// The literal text of one template part, with the delimiters the scanner
// swallowed (`` ` ``, `${`, `}`) stripped back off. Working from the source text
// rather than the cooked value is what keeps the escapes raw.
func rawChunk(text string, node *ast.Node) string {
	start := node.Pos() + 1
	end := node.End() - 1
	if node.Kind == ast.KindTemplateHead || node.Kind == ast.KindTemplateMiddle {
		end = node.End() - 2
	}
	if start > end || start < 0 || end > len(text) {
		return ""
	}
	return strings.ReplaceAll(text[start:end], `\`, `\\`)
}

var NoStringRawRule = rule.Rule{
	Name: "no-string-raw",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		text := ctx.SourceFile.Text()

		sourceOf := func(node *ast.Node) string {
			trimmed := utils.TrimNodeTextRange(ctx.SourceFile, node)
			return text[trimmed.Pos():trimmed.End()]
		}

		return rule.RuleListeners{
			ast.KindTaggedTemplateExpression: func(node *ast.Node) {
				tagged := node.AsTaggedTemplateExpression()
				if !isStringRawTag(tagged.Tag) {
					return
				}

				var rebuilt strings.Builder
				template := tagged.Template
				if ast.IsNoSubstitutionTemplateLiteral(template) {
					rebuilt.WriteString(rawChunk(text, template))
				} else {
					expression := template.AsTemplateExpression()
					rebuilt.WriteString(rawChunk(text, expression.Head))
					for _, span := range expression.TemplateSpans.Nodes {
						templateSpan := span.AsTemplateSpan()
						rebuilt.WriteString("${")
						rebuilt.WriteString(sourceOf(templateSpan.Expression))
						rebuilt.WriteString("}")
						rebuilt.WriteString(rawChunk(text, templateSpan.Literal))
					}
				}

				replacement := "`" + rebuilt.String() + "`"
				ctx.ReportNodeWithFixes(node, buildStringRawMessage(), func() []rule.RuleFix {
					return []rule.RuleFix{rule.RuleFixReplace(ctx.SourceFile, node, replacement)}
				})
			},
		}
	},
}
