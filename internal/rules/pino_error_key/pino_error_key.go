package pino_error_key

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// The key pino serializes Errors under. Anything else is merged in as a plain
// object, and an Error has no enumerable own properties, so it prints as `{}`.
const errorKey = "err"

// pino's level methods. `log` is deliberately absent: it is not a pino level.
var logLevels = map[string]bool{
	"debug": true,
	"error": true,
	"fatal": true,
	"info":  true,
	"trace": true,
	"warn":  true,
}

func buildErrorKeyMessage(name string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "errorKey",
		Description: "`" + name + "` holds an Error but is not pino's `err` key, so it logs as `{}`: pino only serializes Errors under `err`.",
		Help:        "Rename the property to `err` (`{ err: " + name + " }`).",
	}
}

var PinoErrorKeyRule = rule.Rule{
	Name: "pino-error-key",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		isErrorLike := func(t *checker.Type) bool {
			// A union counts when any part is an Error: `Error | undefined`
			// still prints `{}` down the Error branch.
			return utils.SomeUnionTypePart(t, func(part *checker.Type) bool {
				return utils.IsErrorLike(ctx.Program, ctx.TypeChecker, part) ||
					utils.IsReadonlyErrorLike(ctx.Program, ctx.TypeChecker, part)
			})
		}

		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				call := node.AsCallExpression()
				callee := ast.SkipParentheses(call.Expression)
				if !ast.IsPropertyAccessExpression(callee) {
					return
				}
				method := callee.AsPropertyAccessExpression().Name()
				if method == nil || !ast.IsIdentifier(method) || !logLevels[method.Text()] {
					return
				}
				if call.Arguments == nil || len(call.Arguments.Nodes) == 0 {
					return
				}

				// pino only serializes the first argument's own keys.
				first := ast.SkipParentheses(call.Arguments.Nodes[0])
				if !ast.IsObjectLiteralExpression(first) {
					return
				}

				for _, property := range first.AsObjectLiteralExpression().Properties.Nodes {
					var value *ast.Node
					switch {
					case ast.IsPropertyAssignment(property):
						value = property.Initializer()
					case ast.IsShorthandPropertyAssignment(property):
						value = property.Name()
					default:
						// Spreads carry no key to rename, so there is nothing
						// actionable to report.
						continue
					}

					name := property.Name()
					if name == nil || value == nil {
						continue
					}
					// A computed key is not a literal the fix could rename.
					if ast.IsComputedPropertyName(name) {
						continue
					}
					if name.Text() == errorKey {
						continue
					}

					t := utils.GetConstrainedTypeAtLocation(ctx.TypeChecker, value)
					if !isErrorLike(t) {
						continue
					}

					ctx.ReportNode(property, buildErrorKeyMessage(name.Text()))
				}
			},
		}
	},
}
