package require_test_timeout

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

const (
	optionName = "timeout"
	testModule = "node:test"
)

// `it.todo(name)` declares a test that has no body to time out. Every other
// modifier still runs the callback, so it needs the same deadline `it` does.
var exemptModifiers = map[string]bool{
	"todo": true,
}

func buildRequireTestTimeoutMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "requireTestTimeout",
		Description: "`it(...)` must declare a timeout.",
		Help:        "Pass an options object as the second argument: `it('name', { timeout: 5_000 }, async () => {})`.",
	}
}

// The `it` identifier a callee is rooted at, for `it(...)` and for
// `it.only(...)` / `it.skip(...)` / `it.concurrent(...)`, or nil.
//
// `describe` is deliberately absent: it only groups tests, so its own deadline
// is already the sum of the deadlines the tests inside it declare.
func testIdentifier(expression *ast.Node) *ast.Node {
	if ast.IsIdentifier(expression) {
		if expression.Text() != "it" {
			return nil
		}
		return expression
	}
	if !ast.IsPropertyAccessExpression(expression) {
		return nil
	}
	access := expression.AsPropertyAccessExpression()
	if !ast.IsIdentifier(access.Expression) || access.Expression.Text() != "it" {
		return nil
	}
	if exemptModifiers[access.Name().Text()] {
		return nil
	}
	return access.Expression
}

// The specifier a binding was imported from, or "" for anything else. Walking
// up from the declaration covers every import shape without a case per form.
func importModuleSpecifier(declaration *ast.Node) string {
	for node := declaration; node != nil; node = node.Parent {
		switch node.Kind {
		case ast.KindImportDeclaration:
			specifier := node.AsImportDeclaration().ModuleSpecifier
			if specifier != nil && ast.IsStringLiteral(specifier) {
				return specifier.Text()
			}
			return ""
		case ast.KindSourceFile:
			return ""
		}
	}
	return ""
}

// Keeps the rule off a local helper that happens to be called `it`. Types only
// narrow here: an identifier that resolves to nothing is left matched by name,
// so the rule does not go quiet on exactly the untyped files that need it.
func isNodeTestBinding(typeChecker *checker.Checker, identifier *ast.Node) bool {
	symbol := typeChecker.GetSymbolAtLocation(identifier)
	if symbol == nil || len(symbol.Declarations) == 0 {
		return true
	}
	return importModuleSpecifier(symbol.Declarations[0]) == testModule
}

// A function in the options position is the `it(name, fn)` overload — no
// options object was passed at all. Anything else that is not an object
// literal (a variable holding the options, a helper call) is unknowable here,
// so it is left alone rather than guessed at.
func declaresTimeout(node *ast.Node) bool {
	if !ast.IsObjectLiteralExpression(node) {
		return !ast.IsFunctionExpressionOrArrowFunction(node)
	}
	for _, property := range node.AsObjectLiteralExpression().Properties.Nodes {
		// A spread may carry `timeout`; its contents are not visible here.
		if property.Kind == ast.KindSpreadAssignment {
			return true
		}
		if name := property.Name(); name != nil && name.Text() == optionName {
			return true
		}
	}
	return false
}

var RequireTestTimeoutRule = rule.Rule{
	Name: "require-test-timeout",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				call := node.AsCallExpression()
				identifier := testIdentifier(call.Expression)
				if identifier == nil || !isNodeTestBinding(ctx.TypeChecker, identifier) {
					return
				}

				arguments := []*ast.Node{}
				if call.Arguments != nil {
					arguments = call.Arguments.Nodes
				}
				if len(arguments) >= 2 && declaresTimeout(arguments[1]) {
					return
				}

				// Reported on the callee, not the call: the call spans the whole
				// test body, and an editor squiggle over it hides the rest of
				// the file's diagnostics.
				ctx.ReportNode(call.Expression, buildRequireTestTimeoutMessage())
			},
		}
	},
}
