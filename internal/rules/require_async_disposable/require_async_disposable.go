package require_async_disposable

import (
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/scanner"
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

// The leading whitespace on the line an offset sits on, so inserted statements
// line up with their neighbours.
func indentOf(sourceFile *ast.SourceFile, offset int) string {
	text := sourceFile.Text()
	start := offset
	for start > 0 && text[start-1] != '\n' {
		start--
	}
	end := start
	for end < offset && (text[end] == ' ' || text[end] == '\t') {
		end++
	}
	return text[start:end]
}

// Places `text` as the final statement of the block, whether or not the block
// already has one.
func appendClose(sourceFile *ast.SourceFile, body *ast.Node, statements []*ast.Node, text string) rule.RuleFix {
	if len(statements) > 0 {
		last := statements[len(statements)-1]
		start := scanner.SkipTrivia(sourceFile.Text(), last.Pos())
		return rule.RuleFixInsertAfter(last, "\n"+indentOf(sourceFile, start)+text)
	}
	indent := indentOf(sourceFile, scanner.SkipTrivia(sourceFile.Text(), body.Pos()))
	return rule.RuleFixReplaceRange(
		core.NewTextRange(body.End()-1, body.End()-1),
		"\n"+indent+"  "+text+"\n"+indent,
	)
}

// The span to delete when moving a statement: from the end of the previous
// statement (or the block's opening brace) through the end of this one, so the
// line it occupied goes with it.
func cutRange(sourceFile *ast.SourceFile, body *ast.Node, statements []*ast.Node, index int) core.TextRange {
	from := scanner.SkipTrivia(sourceFile.Text(), body.Pos()) + 1
	if index > 0 {
		from = statements[index-1].End()
	}
	return core.NewTextRange(from, statements[index].End())
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
				ctx.ReportNodeWithFixes(member, buildMustCloseMessage(), func() []rule.RuleFix {
					return []rule.RuleFix{appendClose(ctx.SourceFile, body, statements, "await this.close();")}
				})
				return
			}
			if index != len(statements)-1 {
				statement := statements[index]
				text := strings.TrimSpace(ctx.SourceFile.Text()[statement.Pos():statement.End()])
				ctx.ReportNodeWithFixes(statement, buildMustCloseLastMessage(), func() []rule.RuleFix {
					return []rule.RuleFix{
						rule.RuleFixRemoveRange(cutRange(ctx.SourceFile, body, statements, index)),
						appendClose(ctx.SourceFile, body, statements, text),
					}
				})
			}
		}

		return rule.RuleListeners{
			ast.KindClassDeclaration: check,
			ast.KindClassExpression:  check,
		}
	},
}
