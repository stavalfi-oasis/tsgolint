package require_constant_env_key

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildConstantEnvKeyMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "constantEnvKey",
		Description: "Env var names must be a plain string literal: process.env[\"MY_VAR\"] or process.env.MY_VAR.",
		Help:        "A template literal or a variable hides the name from plain text search, so `find . -type f -name '*.ts' -exec grep -l MY_VAR {} +` no longer lists every place the env var is used. Inline the literal name.",
	}
}

// `process.env[...]`, `foo.env[...]` or a bare `env[...]`.
func isEnvObject(expression *ast.Node) bool {
	if ast.IsIdentifier(expression) {
		return expression.Text() == "env"
	}
	if ast.IsPropertyAccessExpression(expression) {
		return expression.AsPropertyAccessExpression().Name().Text() == "env"
	}
	return false
}

var RequireConstantEnvKeyRule = rule.Rule{
	Name: "require-constant-env-key",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindElementAccessExpression: func(node *ast.Node) {
				access := node.AsElementAccessExpression()
				if !isEnvObject(access.Expression) {
					return
				}
				argument := ast.SkipParentheses(access.ArgumentExpression)
				if ast.IsStringLiteral(argument) {
					return
				}
				ctx.ReportNode(access.ArgumentExpression, buildConstantEnvKeyMessage())
			},
		}
	},
}
