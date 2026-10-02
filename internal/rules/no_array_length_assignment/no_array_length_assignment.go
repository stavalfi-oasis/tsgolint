package no_array_length_assignment

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildLengthAssignmentMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "lengthAssignment",
		Description: "Don't assign to `.length`.",
		Help:        "Truncating in place mutates an array every other holder still sees. Assign a fresh array instead: `this.#workers = []`.",
	}
}

// `length` is only writable on an array, so the assignment target is the whole
// signal. Reading the type here would buy nothing and would make the rule go
// quiet whenever the type graph is broken.
func isLengthTarget(node *ast.Node) bool {
	if ast.IsPropertyAccessExpression(node) {
		return node.AsPropertyAccessExpression().Name().Text() == "length"
	}

	if ast.IsElementAccessExpression(node) {
		argument := node.AsElementAccessExpression().ArgumentExpression
		return ast.IsStringLiteralLike(argument) && argument.Text() == "length"
	}

	return false
}

var NoArrayLengthAssignmentRule = rule.Rule{
	Name: "no-array-length-assignment",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindBinaryExpression: func(node *ast.Node) {
				binary := node.AsBinaryExpression()
				if !ast.IsAssignmentOperator(binary.OperatorToken.Kind) || !isLengthTarget(binary.Left) {
					return
				}

				ctx.ReportNode(node, buildLengthAssignmentMessage())
			},
		}
	},
}
