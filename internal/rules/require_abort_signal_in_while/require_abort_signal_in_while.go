package require_abort_signal_in_while

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

const abortSignalTypeName = "AbortSignal"

func buildRequireAbortSignalInWhileMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "requireAbortSignalInWhile",
		Description: "A `while` condition must test an `AbortSignal`.",
		Help:        "Add `!<signal>.aborted &&` to the condition so the loop ends when the signal aborts, and keep the real stop conditions alongside it.",
	}
}

// Whether the receiver is an `AbortSignal`, following unions so
// `AbortSignal | undefined` still counts, and base types so a subclass does.
// The property name alone is not enough — plenty of unrelated objects carry an
// `aborted` flag that nothing ever sets from the outside.
func isAbortSignal(typeChecker *checker.Checker, node *ast.Node) bool {
	t := typeChecker.GetTypeAtLocation(node)
	if t == nil {
		return false
	}

	seen := map[*checker.Type]bool{}
	var walk func(t *checker.Type) bool
	walk = func(t *checker.Type) bool {
		if t == nil || seen[t] {
			return false
		}
		seen[t] = true
		if symbol := checker.Type_symbol(t); symbol != nil && symbol.Name == abortSignalTypeName {
			return true
		}
		for _, part := range utils.UnionTypeParts(t) {
			if part != t && walk(part) {
				return true
			}
		}
		for _, base := range checker.Checker_getBaseTypes(typeChecker, t) {
			if walk(base) {
				return true
			}
		}
		return false
	}
	return walk(t)
}

// `!<signal>.aborted`, which is the one shape that actually ends the loop on
// abort. `signal.aborted` on its own is the inverse test, so it is not enough.
func isNegatedAbortedCheck(typeChecker *checker.Checker, node *ast.Node) bool {
	if node.Kind != ast.KindPrefixUnaryExpression {
		return false
	}
	unary := node.AsPrefixUnaryExpression()
	if unary.Operator != ast.KindExclamationToken {
		return false
	}
	operand := ast.SkipParentheses(unary.Operand)
	if !ast.IsAccessExpression(operand) || accessedName(operand) != "aborted" {
		return false
	}
	return isAbortSignal(typeChecker, operand.Expression())
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
func hasAbortedCheck(typeChecker *checker.Checker, node *ast.Node) bool {
	node = ast.SkipParentheses(node)
	if isNegatedAbortedCheck(typeChecker, node) {
		return true
	}
	found := false
	node.ForEachChild(func(child *ast.Node) bool {
		if hasAbortedCheck(typeChecker, child) {
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
			if !hasAbortedCheck(ctx.TypeChecker, condition) {
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
