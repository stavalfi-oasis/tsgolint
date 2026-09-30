package no_anonymous_functions

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildAnonymousIifeMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "anonymousIife",
		Description: "Anonymous immediately-invoked functions are banned.",
		Help:        "Extract to a named function or class method.",
	}
}

var NoAnonymousFunctionsRule = rule.Rule{
	Name: "no-anonymous-functions",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				// estree has no parenthesized-expression node, so the JS rule saw
				// `(function () {})()` as a bare function callee. The TS AST keeps
				// the parens, hence the skip.
				callee := ast.SkipParentheses(node.AsCallExpression().Expression)
				if ast.IsFunctionExpressionOrArrowFunction(callee) {
					ctx.ReportNode(callee, buildAnonymousIifeMessage())
				}
			},
		}
	},
}
