package no_useless_template_cast

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// A template span already performs ToString on its expression, so a cast adds
// nothing for any type whose ToString is the identity the cast would apply.
//
// `symbol` is deliberately absent: `${sym}` throws a TypeError while
// `String(sym)` works, so there the cast is load-bearing. Objects are absent
// too — both forms print "[object Object]", but no-string-error owns that case
// and its advice (serializeError) is the one worth printing.
const redundantInTemplateFlags = checker.TypeFlagsStringLike |
	checker.TypeFlagsNumberLike |
	checker.TypeFlagsBooleanLike |
	checker.TypeFlagsBigIntLike |
	checker.TypeFlagsNull |
	checker.TypeFlagsUndefined

// `x.toString()` dereferences x, so it is only equivalent to `${x}` when x is
// never nullish.
const toStringReceiverFlags = checker.TypeFlagsStringLike |
	checker.TypeFlagsNumberLike |
	checker.TypeFlagsBooleanLike |
	checker.TypeFlagsBigIntLike

func buildUselessStringCallMessage(expression string, typeName string) rule.RuleMessage {
	return rule.RuleMessage{
		Id: "uselessStringCall",
		Description: "String(" + expression + ") inside a template literal is useless: " + expression +
			" is `" + typeName + "`, and a template literal already converts it to a string by itself.",
		Help: "Write `${" + expression + "}` — the result is byte-for-byte the same string.",
	}
}

func buildUselessToStringMessage(expression string, typeName string) rule.RuleMessage {
	return rule.RuleMessage{
		Id: "uselessToString",
		Description: expression + ".toString() inside a template literal is useless: " + expression +
			" is `" + typeName + "`, and a template literal already converts it to a string by itself.",
		Help: "Write `${" + expression + "}` — the result is byte-for-byte the same string.",
	}
}

var NoUselessTemplateCastRule = rule.Rule{
	Name: "no-useless-template-cast",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		nodeText := func(node *ast.Node) string {
			r := utils.TrimNodeTextRange(ctx.SourceFile, node)
			return ctx.SourceFile.Text()[r.Pos():r.End()]
		}

		// A union is only redundant when every member is — `number | symbol`
		// still needs the cast for the symbol branch.
		isEveryPartFlagged := func(t *checker.Type, flags checker.TypeFlags) bool {
			return !utils.SomeUnionTypePart(t, func(part *checker.Type) bool {
				return !utils.IsTypeFlagSet(part, flags)
			})
		}

		checkExpression := func(expression *ast.Node) {
			if expression.Kind != ast.KindCallExpression {
				return
			}
			call := expression.AsCallExpression()

			// `String(x)`
			if ast.IsIdentifier(call.Expression) && call.Expression.Text() == "String" {
				if call.Arguments == nil || len(call.Arguments.Nodes) != 1 {
					return
				}
				argument := call.Arguments.Nodes[0]
				t := utils.GetConstrainedTypeAtLocation(ctx.TypeChecker, argument)
				if !isEveryPartFlagged(t, redundantInTemplateFlags) {
					return
				}
				ctx.ReportNode(expression, buildUselessStringCallMessage(nodeText(argument), ctx.TypeChecker.TypeToString(t)))
				return
			}

			// `x.toString()` — a radix argument changes the output, so only the
			// no-argument form is redundant.
			if !ast.IsPropertyAccessExpression(call.Expression) {
				return
			}
			access := call.Expression.AsPropertyAccessExpression()
			if access.Name().Text() != "toString" {
				return
			}
			if call.Arguments != nil && len(call.Arguments.Nodes) > 0 {
				return
			}
			// `x?.toString()` yields `undefined` rather than "undefined", so it
			// is not the same string.
			if call.QuestionDotToken != nil || access.QuestionDotToken != nil {
				return
			}

			receiver := access.Expression
			t := utils.GetConstrainedTypeAtLocation(ctx.TypeChecker, receiver)
			if !isEveryPartFlagged(t, toStringReceiverFlags) {
				return
			}
			ctx.ReportNode(expression, buildUselessToStringMessage(nodeText(receiver), ctx.TypeChecker.TypeToString(t)))
		}

		return rule.RuleListeners{
			ast.KindTemplateExpression: func(node *ast.Node) {
				// A tagged template hands the raw values to the tag function,
				// so no conversion happens and the cast may be meaningful.
				if node.Parent != nil && node.Parent.Kind == ast.KindTaggedTemplateExpression {
					return
				}
				for _, span := range node.AsTemplateExpression().TemplateSpans.Nodes {
					if span.Kind != ast.KindTemplateSpan {
						continue
					}
					checkExpression(span.AsTemplateSpan().Expression)
				}
			},
		}
	},
}
