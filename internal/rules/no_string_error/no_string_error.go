package no_string_error

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// Anything String() renders faithfully. Everything else — objects, Errors,
// class instances — comes out as "[object Object]", which is the whole point of
// the rule.
const stringableFlags = checker.TypeFlagsStringLike |
	checker.TypeFlagsNumberLike |
	checker.TypeFlagsBooleanLike |
	checker.TypeFlagsBigIntLike |
	checker.TypeFlagsESSymbolLike |
	checker.TypeFlagsNull |
	checker.TypeFlagsUndefined |
	// `any`/`unknown` carry no information, so reporting them is guesswork.
	checker.TypeFlagsAny |
	checker.TypeFlagsUnknown |
	checker.TypeFlagsNever |
	checker.TypeFlagsTypeParameter

func buildStringErrorMessage(name string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "stringError",
		Description: "String(" + name + ") is banned: it prints [object Object] for anything that is not already a string.",
		Help:        "Use serializeError(" + name + ").message from the serialize-error package instead.",
	}
}

var NoStringErrorRule = rule.Rule{
	Name: "no-string-error",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		// A union is only safe when every member is safe — `string | Error`
		// still prints "[object Object]" down the Error branch.
		isStringable := func(t *checker.Type) bool {
			return !utils.SomeUnionTypePart(t, func(part *checker.Type) bool {
				return !utils.IsTypeFlagSet(part, stringableFlags)
			})
		}

		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				call := node.AsCallExpression()
				if !ast.IsIdentifier(call.Expression) ||
					call.Expression.Text() != "String" ||
					call.Arguments == nil ||
					len(call.Arguments.Nodes) != 1 {
					return
				}

				argument := call.Arguments.Nodes[0]
				t := utils.GetConstrainedTypeAtLocation(ctx.TypeChecker, argument)
				if isStringable(t) {
					return
				}

				ctx.ReportNode(node, buildStringErrorMessage(argument.Text()))
			},
		}
	},
}
