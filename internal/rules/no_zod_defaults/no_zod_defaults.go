package no_zod_defaults

import (
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

var bannedMethods = map[string]struct{}{
	"default":  {},
	"prefault": {},
}

func buildZodDefaultMessage(method string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "zodDefault",
		Description: "zod ." + method + "() is banned: a schema must not invent a value.",
		Help:        "Make the field required and inject it (env var, CLI option), so a missing one fails loudly instead of silently falling back.",
	}
}

// zod's runtime classes are all named ZodSomething (ZodString, ZodObject, ...),
// so the receiver's type symbol identifies a schema no matter what the binding
// is called. The JS rule instead walked the expression back to an identifier
// literally named `z`, which missed every aliased or re-exported schema.
func isZodSchema(typeChecker *checker.Checker, node *ast.Node) bool {
	t := utils.GetConstrainedTypeAtLocation(typeChecker, node)
	return utils.SomeUnionTypePart(t, func(part *checker.Type) bool {
		symbol := checker.Type_symbol(part)
		return symbol != nil && strings.HasPrefix(symbol.Name, "Zod")
	})
}

var NoZodDefaultsRule = rule.Rule{
	Name: "no-zod-defaults",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				call := node.AsCallExpression()
				if !ast.IsPropertyAccessExpression(call.Expression) {
					return
				}

				access := call.Expression.AsPropertyAccessExpression()
				method := access.Name().Text()
				if _, banned := bannedMethods[method]; !banned {
					return
				}

				if !isZodSchema(ctx.TypeChecker, access.Expression) {
					return
				}

				ctx.ReportNode(node, buildZodDefaultMessage(method))
			},
		}
	},
}
