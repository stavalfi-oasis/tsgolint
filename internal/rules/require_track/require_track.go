package require_track

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

const baseClassName = "ADisposable"

func buildRequireTrackMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "requireTrack",
		Description: "This promise is not in the in-flight set, so 'close()' will not wait for it.",
		Help:        "Wrap it in 'this.track(...)'. Allowed bare: 'this.track()', 'this.close()', 'super.close()', and 'await using'.",
	}
}

// Walks the base-class chain, so a class extending a subclass of ADisposable is
// covered. The JS rule compared `superClass.name` to "ADisposable" directly, so
// it only ever saw direct subclasses spelled with that exact identifier.
func extendsDisposable(typeChecker *checker.Checker, class *ast.Node) bool {
	t := typeChecker.GetTypeAtLocation(class)
	if t == nil {
		return false
	}

	seen := map[*checker.Type]bool{}
	var walk func(t *checker.Type) bool
	walk = func(t *checker.Type) bool {
		if t == nil || seen[t] {
			return false
		}
		seen[t] = true
		if symbol := checker.Type_symbol(t); symbol != nil && symbol.Name == baseClassName {
			return true
		}
		for _, base := range checker.Checker_getBaseTypes(typeChecker, t) {
			if walk(base) {
				return true
			}
		}
		return false
	}

	for _, base := range checker.Checker_getBaseTypes(typeChecker, t) {
		if walk(base) {
			return true
		}
	}
	return false
}

// `this.track(...)`, `this.close()` and `super.close()` are the sanctioned bare
// calls — tracking them would either be redundant or deadlock the drain.
func isAllowedCall(node *ast.Node) bool {
	expression := ast.SkipParentheses(node)
	if ast.IsAwaitExpression(expression) {
		expression = ast.SkipParentheses(expression.AsAwaitExpression().Expression)
	}
	if !ast.IsCallExpression(expression) {
		return false
	}
	callee := expression.AsCallExpression().Expression
	if !ast.IsPropertyAccessExpression(callee) {
		return false
	}
	access := callee.AsPropertyAccessExpression()
	receiver := access.Expression
	if receiver.Kind != ast.KindThisKeyword && receiver.Kind != ast.KindSuperKeyword {
		return false
	}
	method := access.Name().Text()
	return method == "track" || method == "close"
}

func enclosingClass(node *ast.Node) *ast.Node {
	for current := node.Parent; current != nil; current = current.Parent {
		if ast.IsClassLike(current) {
			return current
		}
		// A promise created inside a static member has no instance to track on.
		if ast.IsMethodDeclaration(current) || ast.IsPropertyDeclaration(current) {
			if ast.HasSyntacticModifier(current, ast.ModifierFlagsStatic) {
				return nil
			}
		}
	}
	return nil
}

var RequireTrackRule = rule.Rule{
	Name: "require-track",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		isPromise := func(node *ast.Node) bool {
			t := utils.GetConstrainedTypeAtLocation(ctx.TypeChecker, node)
			return utils.IsThenableType(ctx.TypeChecker, node, t)
		}

		// The JS rule listened on AwaitExpression only, so a promise that was
		// never awaited — `this.doWork();` — was invisible to it. Reporting on
		// the expression statement instead catches both, and the type check is
		// what makes that possible: without `await` there is no other signal
		// that the value is a promise.
		check := func(node *ast.Node, expression *ast.Node) {
			if expression == nil || isAllowedCall(expression) {
				return
			}
			class := enclosingClass(node)
			if class == nil || !extendsDisposable(ctx.TypeChecker, class) {
				return
			}
			if !isPromise(expression) {
				return
			}
			ctx.ReportNode(node, buildRequireTrackMessage())
		}

		return rule.RuleListeners{
			ast.KindAwaitExpression: func(node *ast.Node) {
				check(node, node.AsAwaitExpression().Expression)
			},
			ast.KindExpressionStatement: func(node *ast.Node) {
				expression := node.AsExpressionStatement().Expression
				// Awaited ones are handled by the listener above, which reports
				// on the await node itself.
				if ast.IsAwaitExpression(ast.SkipParentheses(expression)) {
					return
				}
				check(node, expression)
			},
		}
	},
}
