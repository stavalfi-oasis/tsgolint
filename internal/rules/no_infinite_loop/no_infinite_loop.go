package no_infinite_loop

import (
	"strconv"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

// A condition that is a literal the language already knows is truthy, so the
// loop states no stop condition: `true`, a non-zero number, a non-empty string.
func isAlwaysTruthyLiteral(node *ast.Node) bool {
	switch node.Kind {
	case ast.KindTrueKeyword:
		return true
	case ast.KindNumericLiteral:
		value, err := strconv.ParseFloat(node.Text(), 64)
		return err == nil && value != 0
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral:
		return node.Text() != ""
	}
	return false
}

func buildInfiniteLoopMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "infiniteLoop",
		Description: "`for (;;)` / `while (true)` is banned.",
		Help:        "Write `while (!signal.aborted)` so Ctrl-C ends the loop, and keep the real stop conditions as `break`s inside. A `for` that still counts (`for (let i = 0; ; i += n)`) is fine.",
	}
}

var NoInfiniteLoopRule = rule.Rule{
	Name: "no-infinite-loop",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindForStatement: func(node *ast.Node) {
				statement := node.AsForStatement()
				if statement.Initializer == nil && statement.Condition == nil && statement.Incrementor == nil {
					ctx.ReportNode(node, buildInfiniteLoopMessage())
				}
			},
			ast.KindWhileStatement: func(node *ast.Node) {
				if isAlwaysTruthyLiteral(ast.SkipParentheses(node.AsWhileStatement().Expression)) {
					ctx.ReportNode(node, buildInfiniteLoopMessage())
				}
			},
		}
	},
}
