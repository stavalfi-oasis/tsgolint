package no_zod_passthrough

import (
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func buildZodPassthroughMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "zodPassthrough",
		Description: "ZodObject.passthrough() is deprecated in zod v4.",
		Help:        "Use z.looseObject({...}) or .loose() instead.",
	}
}

// The JS rule reported any zero-argument `.passthrough()` call, on any receiver.
// Checking the receiver's type keeps it to zod schemas, whose runtime classes
// are all named ZodSomething.
func isZodSchema(typeChecker *checker.Checker, node *ast.Node) bool {
	t := utils.GetConstrainedTypeAtLocation(typeChecker, node)
	return utils.SomeUnionTypePart(t, func(part *checker.Type) bool {
		symbol := checker.Type_symbol(part)
		return symbol != nil && strings.HasPrefix(symbol.Name, "Zod")
	})
}

var NoZodPassthroughRule = rule.Rule{
	Name: "no-zod-passthrough",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				call := node.AsCallExpression()
				if !ast.IsPropertyAccessExpression(call.Expression) {
					return
				}
				if call.Arguments != nil && len(call.Arguments.Nodes) != 0 {
					return
				}

				access := call.Expression.AsPropertyAccessExpression()
				if access.Name().Text() != "passthrough" {
					return
				}

				if !isZodSchema(ctx.TypeChecker, access.Expression) {
					return
				}

				ctx.ReportNode(node, buildZodPassthroughMessage())
			},
		}
	},
}
