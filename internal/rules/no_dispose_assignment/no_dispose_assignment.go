package no_dispose_assignment

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/oasis"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildDisposeAssignmentMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "disposeAssignment",
		Description: "Don't write to an instance property inside `[Symbol.dispose]` / `[Symbol.asyncDispose]`.",
		Help:        "Disposal is the end of the instance's life, so the new value is never read — it only hides use-after-dispose. Copy the field into a local and tear the local down instead.",
	}
}

// Unwraps the layers that can sit between an assignment target and the object
// it is reached through, so `(this.#a!).b = 1` roots at `this` the same way
// `this.#a.b = 1` does.
func rootsAtThis(node *ast.Node) bool {
	for current := node; current != nil; {
		switch {
		case current.Kind == ast.KindThisKeyword:
			return true
		case ast.IsPropertyAccessExpression(current) || ast.IsElementAccessExpression(current) ||
			ast.IsParenthesizedExpression(current) || ast.IsNonNullExpression(current):
			current = current.Expression()
		default:
			return false
		}
	}
	return false
}

// Reports every this-rooted write in a target position. Plain targets are the
// common case; the literal arms cover destructuring, where each element is
// itself a target: `[this.#a] = xs`, `({ b: this.#c } = o)`.
func reportTargets(ctx rule.RuleContext, target *ast.Node) {
	if target == nil {
		return
	}

	switch {
	case ast.IsArrayLiteralExpression(target):
		for _, element := range target.AsArrayLiteralExpression().Elements.Nodes {
			reportTargets(ctx, element)
		}
	case ast.IsObjectLiteralExpression(target):
		for _, property := range target.AsObjectLiteralExpression().Properties.Nodes {
			if ast.IsPropertyAssignment(property) {
				reportTargets(ctx, property.AsPropertyAssignment().Initializer)
			}
		}
	case ast.IsSpreadElement(target):
		reportTargets(ctx, target.AsSpreadElement().Expression)
	case ast.IsBinaryExpression(target) && target.AsBinaryExpression().OperatorToken.Kind == ast.KindEqualsToken:
		// A default inside a pattern: `[this.#a = 1] = xs`.
		reportTargets(ctx, target.AsBinaryExpression().Left)
	case rootsAtThis(target):
		ctx.ReportNode(target, buildDisposeAssignmentMessage())
	}
}

// Walks out to the member the node belongs to. Arrow functions are transparent
// because they share the enclosing `this`; anything that rebinds `this` — a
// `function`, a nested class, a static block — means the write lands on some
// other object and is none of this rule's business.
func isInsideDisposeMember(node *ast.Node) bool {
	for current := node.Parent; current != nil; current = current.Parent {
		switch {
		case ast.IsMethodDeclaration(current):
			return oasis.IsDisposeMember(current)
		case ast.IsFunctionDeclaration(current) || ast.IsFunctionExpression(current) ||
			ast.IsClassLike(current) || ast.IsClassStaticBlockDeclaration(current) ||
			ast.IsConstructorDeclaration(current) || ast.IsAccessor(current) ||
			ast.IsPropertyDeclaration(current):
			return false
		}
	}
	return false
}

var NoDisposeAssignmentRule = rule.Rule{
	Name: "no-dispose-assignment",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindBinaryExpression: func(node *ast.Node) {
				binary := node.AsBinaryExpression()
				if !ast.IsAssignmentOperator(binary.OperatorToken.Kind) || !isInsideDisposeMember(node) {
					return
				}
				reportTargets(ctx, binary.Left)
			},
			ast.KindPrefixUnaryExpression: func(node *ast.Node) {
				unary := node.AsPrefixUnaryExpression()
				if !isIncrementOrDecrement(unary.Operator) || !isInsideDisposeMember(node) {
					return
				}
				if rootsAtThis(unary.Operand) {
					ctx.ReportNode(unary.Operand, buildDisposeAssignmentMessage())
				}
			},
			ast.KindPostfixUnaryExpression: func(node *ast.Node) {
				unary := node.AsPostfixUnaryExpression()
				if !isIncrementOrDecrement(unary.Operator) || !isInsideDisposeMember(node) {
					return
				}
				if rootsAtThis(unary.Operand) {
					ctx.ReportNode(unary.Operand, buildDisposeAssignmentMessage())
				}
			},
			ast.KindDeleteExpression: func(node *ast.Node) {
				operand := node.AsDeleteExpression().Expression
				if !isInsideDisposeMember(node) || !rootsAtThis(operand) {
					return
				}
				ctx.ReportNode(operand, buildDisposeAssignmentMessage())
			},
		}
	},
}

func isIncrementOrDecrement(operator ast.Kind) bool {
	return operator == ast.KindPlusPlusToken || operator == ast.KindMinusMinusToken
}
