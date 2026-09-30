package require_zod_compile

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/oasis"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

// Methods that consume a schema rather than build one — the chain ends there and
// there is nothing left to compile.
var terminalMethods = map[string]bool{
	"decode":         true,
	"decodeAsync":    true,
	"encode":         true,
	"encodeAsync":    true,
	"parse":          true,
	"parseAsync":     true,
	"safeDecode":     true,
	"safeEncode":     true,
	"safeParse":      true,
	"safeParseAsync": true,
}

func buildRequireCompileMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "requireZodCompile",
		Description: "Every zod schema must be AOT-compiled.",
		Help:        "Wrap the whole schema expression in z.compile(...).",
	}
}

func calleeMethod(node *ast.Node) string {
	if !ast.IsCallExpression(node) {
		return ""
	}
	if callee := node.AsCallExpression().Expression; ast.IsPropertyAccessExpression(callee) {
		return callee.AsPropertyAccessExpression().Name().Text()
	}
	return ""
}

var RequireZodCompileRule = rule.Rule{
	Name: "require-zod-compile",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				method := calleeMethod(node)
				if method == "compile" || terminalMethods[method] {
					return
				}

				// The call must produce a schema. This replaces the JS rule's
				// bookkeeping of which local names came from a zod import, so a
				// schema reached through a helper or an alias is still seen.
				if !oasis.IsZodSchemaType(ctx.TypeChecker, node) {
					return
				}

				// Only the outermost schema expression is reported — everything
				// inside it is covered by the same compile() call.
				for parent := node.Parent; parent != nil; parent = parent.Parent {
					if calleeMethod(parent) == "compile" {
						return
					}
					if !ast.IsPropertyAccessExpression(parent) &&
						!ast.IsParenthesizedExpression(parent) &&
						!ast.IsCallExpression(parent) &&
						parent.Kind != ast.KindNonNullExpression {
						break
					}
					if oasis.IsZodSchemaType(ctx.TypeChecker, parent) {
						return
					}
				}

				ctx.ReportNode(node, buildRequireCompileMessage())
			},
		}
	},
}
