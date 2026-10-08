package no_bare_logger_outside_super

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/oasis"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildBareLoggerMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "bareLogger",
		Description: "Use 'this.logger' here, not the constructor's 'logger': the one super() built carries this class's bindings, and the parameter does not, so anything handed the parameter logs under whoever constructed this instance.",
		Help:        "Pass 'this.logger'. The parameter belongs inside the 'super(...)' call only.",
	}
}

// The `logger` parameter of a constructor, whether declared plainly or picked
// out of a destructured options object. Returns nil when there is none.
func loggerParameter(ctor *ast.Node) *ast.Node {
	for _, parameter := range ctor.Parameters() {
		name := parameter.Name()
		if name == nil {
			continue
		}
		if ast.IsIdentifier(name) && name.Text() == "logger" {
			return name
		}
		if !ast.IsObjectBindingPattern(name) {
			continue
		}
		for _, element := range name.AsBindingPattern().Elements.Nodes {
			bound := element.Name()
			if bound != nil && ast.IsIdentifier(bound) && bound.Text() == "logger" {
				return bound
			}
		}
	}
	return nil
}

// The super(...) call inside the constructor body, or nil.
func superCall(ctor *ast.Node) *ast.Node {
	body := ctor.Body()
	if body == nil || !ast.IsBlock(body) {
		return nil
	}
	for _, statement := range body.AsBlock().Statements.Nodes {
		if !ast.IsExpressionStatement(statement) {
			continue
		}
		call := ast.SkipParentheses(statement.AsExpressionStatement().Expression)
		if ast.IsCallExpression(call) && call.AsCallExpression().Expression.Kind == ast.KindSuperKeyword {
			return call
		}
	}
	return nil
}

var NoBareLoggerOutsideSuperRule = rule.Rule{
	Name: "no-bare-logger-outside-super",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		check := func(node *ast.Node) {
			if oasis.IsDisposableBase(node) || !oasis.ExtendsDisposable(ctx.TypeChecker, node) {
				return
			}
			for _, member := range node.Members() {
				if !ast.IsConstructorDeclaration(member) {
					continue
				}
				declaration := loggerParameter(member)
				if declaration == nil {
					continue
				}
				call := superCall(member)
				body := member.Body()
				if body == nil || !ast.IsBlock(body) {
					continue
				}

				var walk func(current *ast.Node) bool
				walk = func(current *ast.Node) bool {
					// Everything inside super(...) is the one place the parameter belongs.
					if call != nil && current == call {
						return false
					}
					if ast.IsIdentifier(current) && current.Text() == "logger" &&
						current != declaration && !isPropertyName(current) {
						ctx.ReportNode(current, buildBareLoggerMessage())
					}
					current.ForEachChild(walk)
					return false
				}
				body.ForEachChild(walk)
			}
		}

		return rule.RuleListeners{
			ast.KindClassDeclaration: check,
			ast.KindClassExpression:  check,
		}
	},
}

// `{ logger: this.logger }` has `logger` as a key, which names nothing.
func isPropertyName(node *ast.Node) bool {
	parent := node.Parent
	if parent == nil {
		return false
	}
	switch {
	case ast.IsPropertyAssignment(parent):
		return parent.Name() == node
	case ast.IsPropertyAccessExpression(parent):
		return parent.AsPropertyAccessExpression().Name() == node
	default:
		return false
	}
}
