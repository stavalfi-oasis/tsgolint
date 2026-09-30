package require_object_params

import (
	"strconv"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildTooManyParamsMessage(label string, count int) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "tooManyParams",
		Description: label + " has " + strconv.Itoa(count) + " parameters.",
		Help:        "Use a single object parameter instead, so every call site names what it passes.",
	}
}

// A `this` parameter is a type annotation, not an argument the caller passes.
func countRealParams(parameters []*ast.Node) int {
	count := 0
	for _, parameter := range parameters {
		name := parameter.Name()
		if name != nil && ast.IsIdentifier(name) && name.Text() == "this" {
			continue
		}
		count++
	}
	return count
}

func nameOf(node *ast.Node) string {
	name := node.Name()
	if name == nil {
		return "anonymous"
	}
	if ast.IsIdentifier(name) || ast.IsPrivateIdentifier(name) || ast.IsStringLiteralLike(name) {
		return name.Text()
	}
	return "anonymous"
}

var RequireObjectParamsRule = rule.Rule{
	Name: "require-object-params",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		report := func(node *ast.Node, label string) {
			count := countRealParams(node.Parameters())
			if count > 1 {
				ctx.ReportNode(node, buildTooManyParamsMessage(label, count))
			}
		}

		// The JS rule only reached an arrow or function expression through a
		// variable declarator, so a callback argument was never checked.
		checkAssigned := func(node *ast.Node) {
			parent := node.Parent
			if parent == nil || !ast.IsVariableDeclaration(parent) {
				return
			}
			report(node, "Function '"+nameOf(parent)+"'")
		}

		return rule.RuleListeners{
			ast.KindArrowFunction: checkAssigned,
			ast.KindConstructor: func(node *ast.Node) {
				report(node, "Constructor")
			},
			ast.KindFunctionDeclaration: func(node *ast.Node) {
				report(node, "Function '"+nameOf(node)+"'")
			},
			ast.KindFunctionExpression: checkAssigned,
			ast.KindMethodDeclaration: func(node *ast.Node) {
				report(node, "Method")
			},
		}
	},
}
