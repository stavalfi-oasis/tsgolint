package require_catch_binding

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildMissingBindingMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "missingCatchBinding",
		Description: "`catch {` is banned.",
		Help:        "Bind the caught value with `catch (error)` so the failure stays inspectable.",
	}
}

var RequireCatchBindingRule = rule.Rule{
	Name: "require-catch-binding",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindCatchClause: func(node *ast.Node) {
				clause := node.AsCatchClause()
				if clause.VariableDeclaration != nil {
					return
				}
				ctx.ReportNodeWithFixes(node, buildMissingBindingMessage(), func() []rule.RuleFix {
					return []rule.RuleFix{
						rule.RuleFixInsertBefore(ctx.SourceFile, clause.Block, "(error) "),
					}
				})
			},
		}
	},
}
