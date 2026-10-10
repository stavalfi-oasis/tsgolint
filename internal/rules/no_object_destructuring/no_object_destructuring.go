package no_object_destructuring

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildObjectDestructuringMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "objectDestructuring",
		Description: "Object destructuring of an already-named value is banned.",
		Help:        "Read through the owner instead: `testHelper.etl.repo(name)`, `this.logger`. Destructuring a call result (`const { tenantId } = await testHelper.newTenant()`), parameters and `for (const { x } of xs)` stay allowed.",
	}
}

// Drops the layers that only re-type or re-group a value, so `(testHelper!)`
// and `testHelper as TestHelper` root at the same identifier `testHelper` does.
func unwrap(node *ast.Node) *ast.Node {
	for current := node; current != nil; {
		switch {
		case ast.IsParenthesizedExpression(current), ast.IsNonNullExpression(current),
			ast.IsAsExpression(current), ast.IsSatisfiesExpression(current),
			ast.IsTypeAssertion(current):
			current = current.Expression()
		default:
			return current
		}
	}
	return nil
}

// Whether a call runs anywhere inside the subtree. A call means the value is
// produced here rather than already sitting under a name, which is the whole
// distinction this rule draws.
func containsCall(node *ast.Node) bool {
	if node == nil {
		return false
	}
	if ast.IsCallExpression(node) || ast.IsNewExpression(node) || ast.IsTaggedTemplateExpression(node) {
		return true
	}
	found := false
	var walk ast.Visitor
	walk = func(child *ast.Node) bool {
		if found {
			return true
		}
		if ast.IsCallExpression(child) || ast.IsNewExpression(child) || ast.IsTaggedTemplateExpression(child) {
			found = true
			return true
		}
		child.ForEachChild(walk)
		return false
	}
	node.ForEachChild(walk)
	return found
}

// A reference chain that only reads names already in scope: `testHelper`,
// `this`, `testHelper.etl`, `helpers[0].etl`. Anything that computes its value
// — a call, `a ?? b`, a ternary — is not one.
func isNamedReference(node *ast.Node) bool {
	for current := unwrap(node); current != nil; {
		switch {
		case current.Kind == ast.KindIdentifier, current.Kind == ast.KindThisKeyword,
			current.Kind == ast.KindSuperKeyword:
			return true
		case ast.IsPropertyAccessExpression(current):
			current = unwrap(current.Expression())
		case ast.IsElementAccessExpression(current):
			if containsCall(current.AsElementAccessExpression().ArgumentExpression) {
				return false
			}
			current = unwrap(current.Expression())
		default:
			return false
		}
	}
	return false
}

var NoObjectDestructuringRule = rule.Rule{
	Name: "no-object-destructuring",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			// Only the pattern a declaration binds directly. Patterns nested inside
			// another pattern ride on their root's verdict, and a parameter's
			// pattern never reaches here with a VariableDeclaration parent.
			ast.KindObjectBindingPattern: func(node *ast.Node) {
				declaration := node.Parent
				if declaration == nil || declaration.Kind != ast.KindVariableDeclaration {
					return
				}

				list := declaration.Parent
				if list == nil {
					return
				}
				if list.Kind == ast.KindCatchClause {
					ctx.ReportNode(node, buildObjectDestructuringMessage())
					return
				}
				if list.Kind == ast.KindVariableDeclarationList && list.Parent != nil &&
					(list.Parent.Kind == ast.KindForOfStatement || list.Parent.Kind == ast.KindForInStatement) {
					return
				}

				if isNamedReference(declaration.AsVariableDeclaration().Initializer) {
					ctx.ReportNode(node, buildObjectDestructuringMessage())
				}
			},
			// The assignment form `({ a } = b)`, which binds nothing new and reads
			// as a block to anyone skimming it.
			ast.KindBinaryExpression: func(node *ast.Node) {
				binary := node.AsBinaryExpression()
				if binary.OperatorToken.Kind != ast.KindEqualsToken {
					return
				}
				if target := ast.SkipParentheses(binary.Left); ast.IsObjectLiteralExpression(target) {
					ctx.ReportNode(target, buildObjectDestructuringMessage())
				}
			},
		}
	},
}
