package no_tracked_close

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func buildTrackedCloseMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "trackedClose",
		Description: "'close()' drains the in-flight set, so tracking it deadlocks the wait.",
		Help:        "Await it directly: 'await this.close()'.",
	}
}

// `this.track(...)` — the method that registers a promise in the in-flight set.
func isTrackCall(node *ast.Node) bool {
	callee := node.AsCallExpression().Expression
	if !ast.IsPropertyAccessExpression(callee) {
		return false
	}
	access := callee.AsPropertyAccessExpression()
	return access.Expression.Kind == ast.KindThisKeyword && access.Name().Text() == "track"
}

// `this.close()` or `super.close()`.
func isCloseCall(node *ast.Node) bool {
	if !ast.IsCallExpression(node) {
		return false
	}
	callee := node.AsCallExpression().Expression
	if !ast.IsPropertyAccessExpression(callee) {
		return false
	}
	access := callee.AsPropertyAccessExpression()
	receiver := access.Expression
	return (receiver.Kind == ast.KindThisKeyword || receiver.Kind == ast.KindSuperKeyword) &&
		access.Name().Text() == "close"
}

var NoTrackedCloseRule = rule.Rule{
	Name: "no-tracked-close",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				if !isTrackCall(node) {
					return
				}
				args := node.AsCallExpression().Arguments
				if args == nil || len(args.Nodes) != 1 {
					return
				}
				argument := args.Nodes[0]
				if !isCloseCall(argument) {
					return
				}
				ctx.ReportNodeWithFixes(node, buildTrackedCloseMessage(), func() []rule.RuleFix {
					text := utils.TrimNodeTextRange(ctx.SourceFile, argument)
					return []rule.RuleFix{
						rule.RuleFixReplace(ctx.SourceFile, node, ctx.SourceFile.Text()[text.Pos():text.End()]),
					}
				})
			},
		}
	},
}
