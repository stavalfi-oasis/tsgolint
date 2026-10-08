package require_logger_child_in_super

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/oasis"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildMissingChildMessage(class string) rule.RuleMessage {
	return rule.RuleMessage{
		Id: "missingLoggerChild",
		Description: "'super()' must be passed 'logger: logger.child({ name: " + class +
			".name })': without the child binding every line this instance logs is labelled by whoever constructed it.",
		Help: "Pass 'logger: logger.child({ name: " + class + ".name })'. Extra bindings, such as 'service', are fine.",
	}
}

// The object literal a call was given as its single argument, or nil.
func singleObjectArgument(call *ast.Node) *ast.Node {
	args := call.AsCallExpression().Arguments
	if args == nil || len(args.Nodes) != 1 {
		return nil
	}
	argument := ast.SkipParentheses(args.Nodes[0])
	if !ast.IsObjectLiteralExpression(argument) {
		return nil
	}
	return argument
}

// The value assigned to `key` in an object literal, or nil. Shorthand counts:
// `{ logger }` yields the identifier itself.
func propertyValue(object *ast.Node, key string) *ast.Node {
	for _, property := range object.AsObjectLiteralExpression().Properties.Nodes {
		name := property.Name()
		if name == nil || !ast.IsIdentifier(name) || name.Text() != key {
			continue
		}
		if ast.IsShorthandPropertyAssignment(property) {
			return name
		}
		if ast.IsPropertyAssignment(property) {
			return ast.SkipParentheses(property.AsPropertyAssignment().Initializer)
		}
	}
	return nil
}

// Reports whether the expression is an `x.child({ name: ... })` call. The
// receiver is deliberately not checked: it is the constructor's logger
// parameter in every real case, and no-bare-logger-outside-super is what pins
// that down.
func isChildCallWithName(expression *ast.Node) bool {
	if expression == nil || !ast.IsCallExpression(expression) {
		return false
	}
	callee := expression.AsCallExpression().Expression
	if !ast.IsPropertyAccessExpression(callee) ||
		callee.AsPropertyAccessExpression().Name().Text() != "child" {
		return false
	}
	bindings := singleObjectArgument(expression)
	return bindings != nil && propertyValue(bindings, "name") != nil
}

var RequireLoggerChildInSuperRule = rule.Rule{
	Name: "require-logger-child-in-super",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		check := func(node *ast.Node) {
			if oasis.IsDisposableBase(node) || !oasis.ExtendsDisposable(ctx.TypeChecker, node) {
				return
			}

			className := "<class>"
			if name := node.Name(); name != nil {
				className = name.Text()
			}

			for _, member := range node.Members() {
				if !ast.IsConstructorDeclaration(member) {
					continue
				}
				body := member.Body()
				if body == nil || !ast.IsBlock(body) {
					continue
				}
				for _, statement := range body.AsBlock().Statements.Nodes {
					if !ast.IsExpressionStatement(statement) {
						continue
					}
					call := ast.SkipParentheses(statement.AsExpressionStatement().Expression)
					if !ast.IsCallExpression(call) ||
						call.AsCallExpression().Expression.Kind != ast.KindSuperKeyword {
						continue
					}
					arguments := singleObjectArgument(call)
					if arguments == nil || !isChildCallWithName(propertyValue(arguments, "logger")) {
						ctx.ReportNode(call, buildMissingChildMessage(className))
					}
					return
				}
			}
		}

		return rule.RuleListeners{
			ast.KindClassDeclaration: check,
			ast.KindClassExpression:  check,
		}
	},
}
