package require_async_disposable

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/typescript-eslint/tsgolint/internal/oasis"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
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
		Help:        "Add 'await this.close();' to the method.",
	}
}

func buildCloseOnceMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "closeOnce",
		Description: "'[Symbol.asyncDispose]' must call 'this.close()' exactly once: the second call throws, because close() rejects a disposal that already happened.",
		Help:        "Delete this call and keep the first one.",
	}
}

func buildNoTrackMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "noTrack",
		Description: "'this.track()' does nothing inside '[Symbol.asyncDispose]': the await already waits for the promise, and after close() the in-flight set is cleared, so nothing will ever drain it.",
		Help:        "Await the promise directly, without wrapping it in track().",
	}
}

// A call to `close`/`track` on `this` or `super`, as in `this.close()` or
// `super.track(x)`. Returns the call node, or nil.
func selfCall(node *ast.Node, method string) *ast.Node {
	call := ast.SkipParentheses(node)
	if !ast.IsCallExpression(call) {
		return nil
	}
	callee := call.AsCallExpression().Expression
	if !ast.IsPropertyAccessExpression(callee) {
		return nil
	}
	access := callee.AsPropertyAccessExpression()
	receiver := access.Expression
	if receiver.Kind != ast.KindThisKeyword && receiver.Kind != ast.KindSuperKeyword {
		return nil
	}
	if access.Name().Text() != method {
		return nil
	}
	return call
}

// Every `this.close()` / `this.track()` call anywhere in the dispose body,
// in source order. Nested functions count: a close hidden in a callback is
// still a close this method is responsible for.
func selfCalls(body *ast.Node, method string) []*ast.Node {
	var found []*ast.Node
	var walk func(node *ast.Node) bool
	walk = func(node *ast.Node) bool {
		if call := selfCall(node, method); call != nil {
			found = append(found, call)
		}
		node.ForEachChild(walk)
		return false
	}
	body.ForEachChild(walk)
	return found
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

// The span to delete when removing a statement: from the end of the previous
// statement (or the block's opening brace) through the end of this one, so the
// line it occupied goes with it. Returns false when the node is not directly
// inside the dispose block — only whole statements are safe to delete.
func cutStatement(sourceFile *ast.SourceFile, body *ast.Node, node *ast.Node) (core.TextRange, bool) {
	statements := body.AsBlock().Statements.Nodes
	for index, statement := range statements {
		if statement.Pos() > node.Pos() || node.End() > statement.End() {
			continue
		}
		from := scanner.SkipTrivia(sourceFile.Text(), body.Pos()) + 1
		if index > 0 {
			from = statements[index-1].End()
		}
		return core.NewTextRange(from, statement.End()), true
	}
	return core.TextRange{}, false
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

			// track() is never right here: the dispose body awaits its own work,
			// and close() clears the set that track() feeds.
			for _, call := range selfCalls(body, "track") {
				ctx.ReportNodeWithFixes(call, buildNoTrackMessage(), func() []rule.RuleFix {
					args := call.AsCallExpression().Arguments
					if args == nil || len(args.Nodes) != 1 {
						return nil
					}
					inner := utils.TrimNodeTextRange(ctx.SourceFile, args.Nodes[0])
					return []rule.RuleFix{
						rule.RuleFixReplace(ctx.SourceFile, call, ctx.SourceFile.Text()[inner.Pos():inner.End()]),
					}
				})
			}

			closes := selfCalls(body, "close")
			if len(closes) == 0 {
				ctx.ReportNodeWithFixes(member, buildMustCloseMessage(), func() []rule.RuleFix {
					return []rule.RuleFix{appendClose(ctx.SourceFile, body, statements, "await this.close();")}
				})
				return
			}
			// Position is deliberately not checked: what has to hold is that the
			// abort fires once, not that it fires on the final line. A dispose
			// that stops its workers, closes, then frees a connection is correct.
			for _, extra := range closes[1:] {
				ctx.ReportNodeWithFixes(extra, buildCloseOnceMessage(), func() []rule.RuleFix {
					cut, ok := cutStatement(ctx.SourceFile, body, extra)
					if !ok {
						return nil
					}
					return []rule.RuleFix{rule.RuleFixRemoveRange(cut)}
				})
			}
		}

		return rule.RuleListeners{
			ast.KindClassDeclaration: check,
			ast.KindClassExpression:  check,
		}
	},
}
