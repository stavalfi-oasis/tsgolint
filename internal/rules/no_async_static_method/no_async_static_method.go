package no_async_static_method

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildAsyncStaticMethodMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "asyncStaticMethod",
		Description: "An async `static` method has no `this`, so the promises it awaits cannot be tracked.",
		Help:        "Drop `static` and make it an instance method, so every promise inside it can go through `this.track(...)` and be drained by `close()`.",
	}
}

// An async arrow or function expression assigned to a static property is the
// same thing written differently — `static run = async () => {}` has no usable
// `this` either, and the rule would be trivial to sidestep without this.
func isAsyncFunctionInitializer(initializer *ast.Node) bool {
	if initializer == nil {
		return false
	}
	expression := ast.SkipParentheses(initializer)
	if !ast.IsArrowFunction(expression) && !ast.IsFunctionExpression(expression) {
		return false
	}
	return ast.HasSyntacticModifier(expression, ast.ModifierFlagsAsync)
}

var NoAsyncStaticMethodRule = rule.Rule{
	Name: "no-async-static-method",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindMethodDeclaration: func(node *ast.Node) {
				if ast.HasSyntacticModifier(node, ast.ModifierFlagsStatic) &&
					ast.HasSyntacticModifier(node, ast.ModifierFlagsAsync) {
					ctx.ReportNode(node, buildAsyncStaticMethodMessage())
				}
			},
			ast.KindPropertyDeclaration: func(node *ast.Node) {
				if !ast.HasSyntacticModifier(node, ast.ModifierFlagsStatic) {
					return
				}
				if isAsyncFunctionInitializer(node.AsPropertyDeclaration().Initializer) {
					ctx.ReportNode(node, buildAsyncStaticMethodMessage())
				}
			},
		}
	},
}
