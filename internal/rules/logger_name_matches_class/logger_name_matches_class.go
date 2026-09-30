package logger_name_matches_class

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

const nameKey = "name"

func buildLoggerNameMessage(expected string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "loggerNameMismatch",
		Description: "The logger child '" + nameKey + "' must be '" + expected + "'.",
		Help:        "Name the logger after the class that owns it, so a log line says where it came from.",
	}
}

// The `name` property of a `<logger>.child({ name: ... })` argument, or nil.
func childNameProperty(node *ast.Node) *ast.Node {
	call := node.AsCallExpression()
	if !ast.IsPropertyAccessExpression(call.Expression) {
		return nil
	}
	if call.Expression.AsPropertyAccessExpression().Name().Text() != "child" {
		return nil
	}
	if call.Arguments == nil || len(call.Arguments.Nodes) == 0 {
		return nil
	}
	argument := call.Arguments.Nodes[0]
	if !ast.IsObjectLiteralExpression(argument) {
		return nil
	}
	for _, property := range argument.AsObjectLiteralExpression().Properties.Nodes {
		if !ast.IsPropertyAssignment(property) {
			continue
		}
		key := property.Name()
		if key == nil {
			continue
		}
		if (ast.IsIdentifier(key) || ast.IsStringLiteralLike(key)) && key.Text() == nameKey {
			return property
		}
	}
	return nil
}

var LoggerNameMatchesClassRule = rule.Rule{
	Name: "logger-name-matches-class",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		text := ctx.SourceFile.Text()

		check := func(node *ast.Node) {
			className := node.Name()
			if className == nil {
				return
			}
			var constructor *ast.Node
			for _, member := range node.Members() {
				if member.Kind == ast.KindConstructor {
					constructor = member
					break
				}
			}
			if constructor == nil {
				return
			}

			expected := className.Text() + "." + nameKey

			var walk func(current *ast.Node) bool
			walk = func(current *ast.Node) bool {
				if ast.IsCallExpression(current) {
					if property := childNameProperty(current); property != nil {
						value := property.AsPropertyAssignment().Initializer
						trimmed := utils.TrimNodeTextRange(ctx.SourceFile, value)
						if text[trimmed.Pos():trimmed.End()] != expected {
							ctx.ReportNodeWithFixes(value, buildLoggerNameMessage(expected), func() []rule.RuleFix {
								return []rule.RuleFix{rule.RuleFixReplace(ctx.SourceFile, value, expected)}
							})
						}
					}
				}
				ast.ForEachChildAndJSDoc(current, ctx.SourceFile, walk)
				return false
			}
			walk(constructor)
		}

		return rule.RuleListeners{
			ast.KindClassDeclaration: check,
			ast.KindClassExpression:  check,
		}
	},
}
