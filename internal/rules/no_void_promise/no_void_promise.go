package no_void_promise

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func buildVoidPromiseMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "voidPromise",
		Description: "`void` on a promise is banned.",
		Help:        "Await it, or attach an explicit .catch() handler.",
	}
}

var NoVoidPromiseRule = rule.Rule{
	Name: "no-void-promise",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			// The JS rule walked the whole program collecting the names of async
			// functions, then guessed from the callee name whether the voided
			// expression was a promise. The type checker answers that directly,
			// so aliased, re-exported and higher-order promise sources are all
			// caught and non-promise `void` expressions are left alone.
			ast.KindVoidExpression: func(node *ast.Node) {
				argument := node.AsVoidExpression().Expression
				if !utils.IsThenableType(ctx.TypeChecker, argument, nil) {
					return
				}

				ctx.ReportNode(node, buildVoidPromiseMessage())
			},
		}
	},
}
