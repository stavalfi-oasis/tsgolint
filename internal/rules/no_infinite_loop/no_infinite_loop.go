package no_infinite_loop

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

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
				if ast.SkipParentheses(node.AsWhileStatement().Expression).Kind == ast.KindTrueKeyword {
					ctx.ReportNode(node, buildInfiniteLoopMessage())
				}
			},
		}
	},
}
