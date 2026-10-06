package require_abort_signal_in_while

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildRequireAbortSignalInWhileMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "requireAbortSignalInWhile",
		Description: "A `while` condition must test an abort signal.",
		Help:        "Add `!<signal>.aborted &&` to the condition so the loop ends when the signal aborts, and keep the real stop conditions alongside it.",
	}
}

// `!<anything>.aborted`, which is the one shape that actually ends the loop on
// abort. `signal.aborted` on its own is the inverse test, so it is not enough.
func isNegatedAbortedCheck(node *ast.Node) bool {
	if node.Kind != ast.KindPrefixUnaryExpression {
		return false
	}
	unary := node.AsPrefixUnaryExpression()
	if unary.Operator != ast.KindExclamationToken {
		return false
	}
	operand := ast.SkipParentheses(unary.Operand)
	return ast.IsAccessExpression(operand) && accessedName(operand) == "aborted"
}

func accessedName(node *ast.Node) string {
	if ast.IsPropertyAccessExpression(node) {
		return node.AsPropertyAccessExpression().Name().Text()
	}
	if ast.IsElementAccessExpression(node) {
		argument := ast.SkipParentheses(node.AsElementAccessExpression().ArgumentExpression)
		if ast.IsStringLiteralLike(argument) {
			return argument.Text()
		}
	}
	return ""
}

// The condition is free to carry its own stop conditions next to the signal
// check, so the check only has to appear somewhere in the condition —
// `!signal.aborted && taken < pending.length` is the common shape.
func hasAbortedCheck(node *ast.Node) bool {
	node = ast.SkipParentheses(node)
	if isNegatedAbortedCheck(node) {
		return true
	}
	found := false
	node.ForEachChild(func(child *ast.Node) bool {
		if hasAbortedCheck(child) {
			found = true
			return true
		}
		return false
	})
	return found
}

var RequireAbortSignalInWhileRule = rule.Rule{
	Name: "require-abort-signal-in-while",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		report := func(node *ast.Node, condition *ast.Node) {
			if !hasAbortedCheck(condition) {
				ctx.ReportNode(node, buildRequireAbortSignalInWhileMessage())
			}
		}
		return rule.RuleListeners{
			ast.KindDoStatement: func(node *ast.Node) {
				report(node, node.AsDoStatement().Expression)
			},
			ast.KindWhileStatement: func(node *ast.Node) {
				report(node, node.AsWhileStatement().Expression)
			},
		}
	},
}
