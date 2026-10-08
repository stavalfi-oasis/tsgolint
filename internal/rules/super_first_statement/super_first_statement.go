package super_first_statement

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildNotFirstMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "superNotFirst",
		Description: "'super()' must be the first statement of the constructor: until it has run 'this' does not exist, so anything above it cannot reach the base class's state.",
		Help:        "Move 'super()' to the top of the constructor.",
	}
}

func isSuperCall(statement *ast.Node) bool {
	if !ast.IsExpressionStatement(statement) {
		return false
	}
	call := ast.SkipParentheses(statement.AsExpressionStatement().Expression)
	return ast.IsCallExpression(call) &&
		call.AsCallExpression().Expression.Kind == ast.KindSuperKeyword
}

var SuperFirstStatementRule = rule.Rule{
	Name: "super-first-statement",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindConstructor: func(node *ast.Node) {
				body := node.Body()
				if body == nil || !ast.IsBlock(body) {
					return
				}
				statements := body.AsBlock().Statements.Nodes
				for index, statement := range statements {
					if !isSuperCall(statement) {
						continue
					}
					if index != 0 {
						ctx.ReportNode(statement, buildNotFirstMessage())
					}
					return
				}
			},
		}
	},
}
