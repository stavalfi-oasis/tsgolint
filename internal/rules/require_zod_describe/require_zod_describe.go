package require_zod_describe

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/oasis"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildRequireDescribeMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "requireZodDescribe",
		Description: "Every root zod schema must have a description.",
		Help:        "Add .describe(\"...\") to the schema chain inside z.compile(...).",
	}
}

// Walks the schema chain inside z.compile(...) back to where it started, looking
// for a .describe(...) link anywhere along it. Nested members are not walked:
// only the root schema needs the description.
func chainHasDescribe(node *ast.Node) bool {
	current := node
	for {
		switch {
		case ast.IsCallExpression(current):
			callee := current.AsCallExpression().Expression
			if ast.IsPropertyAccessExpression(callee) && callee.AsPropertyAccessExpression().Name().Text() == "describe" {
				return true
			}
			current = callee
		case ast.IsPropertyAccessExpression(current):
			current = current.AsPropertyAccessExpression().Expression
		case ast.IsParenthesizedExpression(current):
			current = current.AsParenthesizedExpression().Expression
		case current.Kind == ast.KindNonNullExpression:
			current = current.AsNonNullExpression().Expression
		default:
			return false
		}
	}
}

// The compile() of the zod namespace — the receiver must have a compile member,
// so an unrelated `regex.compile(x)` is not mistaken for it.
func isZodCompileCall(typeChecker *checker.Checker, call *ast.CallExpression) bool {
	if !ast.IsPropertyAccessExpression(call.Expression) {
		return false
	}
	access := call.Expression.AsPropertyAccessExpression()
	if access.Name().Text() != "compile" {
		return false
	}
	t := typeChecker.GetTypeAtLocation(access.Expression)
	return t != nil && checker.Checker_getPropertyOfType(typeChecker, t, "compile") != nil
}

var RequireZodDescribeRule = rule.Rule{
	Name: "require-zod-describe",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				call := node.AsCallExpression()
				if call.Arguments == nil || len(call.Arguments.Nodes) != 1 {
					return
				}
				if !isZodCompileCall(ctx.TypeChecker, call) {
					return
				}

				schema := call.Arguments.Nodes[0]
				if !oasis.IsZodSchemaType(ctx.TypeChecker, schema) {
					return
				}
				if chainHasDescribe(schema) {
					return
				}

				ctx.ReportNode(schema, buildRequireDescribeMessage())
			},
		}
	},
}
