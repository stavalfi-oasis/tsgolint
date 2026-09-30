package require_async_disposable

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/oasis"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildMustExtendMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "mustExtend",
		Description: "Class must extend " + oasis.DisposableBaseName + ": it owns the AbortController and the in-flight set.",
		Help:        "Extend " + oasis.DisposableBaseName + ", or a class that already does.",
	}
}

func buildMustCloseMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "mustClose",
		Description: "'[Symbol.asyncDispose]' must 'await this.close()' so the AbortController is aborted and in-flight work settles.",
		Help:        "Add 'await this.close();' as the last statement.",
	}
}

func buildMustCloseLastMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "mustCloseLast",
		Description: "'await this.close()' must be the last statement of '[Symbol.asyncDispose]' so the AbortController is aborted only after everything else settled.",
		Help:        "Move it to the end of the method.",
	}
}

// Matches `await this.close()` and `await this.track(this.close())`, on `this`
// or `super`.
func isCloseStatement(statement *ast.Node) bool {
	if !ast.IsExpressionStatement(statement) {
		return false
	}
	expression := ast.SkipParentheses(statement.AsExpressionStatement().Expression)
	if !ast.IsAwaitExpression(expression) {
		return false
	}

	call := ast.SkipParentheses(expression.AsAwaitExpression().Expression)
	if !ast.IsCallExpression(call) {
		return false
	}

	// Unwrap one layer of this.track(...).
	if callee := call.AsCallExpression().Expression; ast.IsPropertyAccessExpression(callee) &&
		callee.AsPropertyAccessExpression().Name().Text() == "track" {
		args := call.AsCallExpression().Arguments
		if args != nil && len(args.Nodes) == 1 && ast.IsCallExpression(ast.SkipParentheses(args.Nodes[0])) {
			call = ast.SkipParentheses(args.Nodes[0])
		}
	}

	callee := call.AsCallExpression().Expression
	if !ast.IsPropertyAccessExpression(callee) {
		return false
	}
	access := callee.AsPropertyAccessExpression()
	receiver := access.Expression
	return (receiver.Kind == ast.KindThisKeyword || receiver.Kind == ast.KindSuperKeyword) &&
		access.Name().Text() == "close"
}

var RequireAsyncDisposableRule = rule.Rule{
	Name: "require-async-disposable",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		check := func(node *ast.Node) {
			if oasis.IsDisposableBase(node) {
				return
			}

			// The type check is what makes the transitive case work: a class
			// extending a subclass of ADisposable already owns an AbortController,
			// but the JS rule's superclass-name match reported it anyway.
			if !oasis.ExtendsDisposable(ctx.TypeChecker, node) {
				name := node.Name()
				if name == nil {
					name = node
				}
				ctx.ReportNode(name, buildMustExtendMessage())
				return
			}

			var member *ast.Node
			for _, candidate := range node.Members() {
				if oasis.IsAsyncDisposeMember(candidate) {
					member = candidate
					break
				}
			}
			if member == nil {
				return
			}

			body := member.Body()
			if body == nil || !ast.IsBlock(body) {
				return
			}
			statements := body.AsBlock().Statements.Nodes

			index := -1
			for i, statement := range statements {
				if isCloseStatement(statement) {
					index = i
					break
				}
			}
			if index == -1 {
				ctx.ReportNode(member, buildMustCloseMessage())
				return
			}
			if index != len(statements)-1 {
				ctx.ReportNode(statements[index], buildMustCloseLastMessage())
			}
		}

		return rule.RuleListeners{
			ast.KindClassDeclaration: check,
			ast.KindClassExpression:  check,
		}
	},
}
