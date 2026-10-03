package logger_name_matches_class

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

const nameKey = "name"

const loggerKey = "logger"

func buildLoggerNameMessage(expected string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "loggerNameMismatch",
		Description: "The logger child '" + nameKey + "' must be '" + expected + "'.",
		Help:        "Name the logger after the class that owns it, so a log line says where it came from.",
	}
}

func buildMissingChildMessage(expected string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "loggerChildMissing",
		Description: "A constructor that accepts a '" + loggerKey + "' must create a child logger named '" + expected + "'.",
		Help:        "Call '.child({ " + nameKey + ": " + expected + " })' on the logger in the constructor, so every log line this class writes says where it came from.",
	}
}

// The constructor parameter that carries a logger — either a `logger`
// parameter of its own, or an options object with a `logger` member, the shape
// nearly every class in the repo uses — or nil when it is handed none.
func loggerParam(typeChecker *checker.Checker, constructor *ast.Node) *ast.Node {
	for _, parameter := range constructor.Parameters() {
		name := parameter.Name()
		if name != nil && ast.IsIdentifier(name) && name.Text() == loggerKey {
			return parameter
		}
		if name != nil && ast.IsObjectBindingPattern(name) {
			for _, element := range name.AsBindingPattern().Elements.Nodes {
				binding := element.AsBindingElement()
				property := binding.PropertyName
				if property == nil {
					property = binding.Name()
				}
				if property != nil && ast.IsIdentifier(property) && property.Text() == loggerKey {
					return parameter
				}
			}
		}
		// Asking the checker rather than reading the annotation is what covers a
		// named options interface and an intersection, neither of which spells
		// the member on the parameter.
		t := utils.GetConstrainedTypeAtLocation(typeChecker, parameter)
		if t != nil && checker.Checker_getPropertyOfType(typeChecker, t, loggerKey) != nil {
			return parameter
		}
	}
	return nil
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
			named := false

			var walk func(current *ast.Node) bool
			walk = func(current *ast.Node) bool {
				if ast.IsCallExpression(current) {
					if property := childNameProperty(current); property != nil {
						named = true
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

			// A named child anywhere in the constructor is enough: a wrong name
			// is already reported above, and reporting the same constructor
			// twice for one mistake would be noise.
			if !named {
				if parameter := loggerParam(ctx.TypeChecker, constructor); parameter != nil {
					ctx.ReportNode(parameter, buildMissingChildMessage(expected))
				}
			}
		}

		return rule.RuleListeners{
			ast.KindClassDeclaration: check,
			ast.KindClassExpression:  check,
		}
	},
}
