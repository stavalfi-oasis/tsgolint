package new_only_in_constructor

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

type NewOnlyInConstructorOptions struct {
	Allow []string `json:"allow"`
}

func buildNewOutsideConstructorMessage(name string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "newOutsideConstructor",
		Description: "`new " + name + "` runs outside the constructor, so the field holding it is undefined until this method runs.",
		Help:        "Construct it in the constructor and pass whatever is only known later as an argument to this method.",
	}
}

// The name written after `new`: `new Foo()` and `new ns.Foo()` both report `Foo`.
func constructedName(node *ast.Node) string {
	expression := ast.SkipParentheses(node.AsNewExpression().Expression)
	if ast.IsIdentifier(expression) {
		return expression.Text()
	}
	if ast.IsPropertyAccessExpression(expression) {
		return expression.AsPropertyAccessExpression().Name().Text()
	}
	return ""
}

// Walks out of `node` to the first thing that decides whether this `new` is in a
// constructor: the constructor itself, any other class member, or the class. A
// `new` in a plain function or at module scope reaches neither and is left alone -
// the bug this rule exists for is a class field built in `start()` and read in
// `[Symbol.asyncDispose]()`, not construction in ordinary code.
// `Logger.child()` returning `new Logger(...)` is a class making another of itself, not a class
// holding a dependency - there is no field to be left undefined, so the rule has nothing to say.
func isEnclosingClassItself(node *ast.Node, name string) bool {
	for current := node.Parent; current != nil; current = current.Parent {
		if ast.IsClassLike(current) {
			className := current.Name()
			return className != nil && className.Text() == name
		}
	}
	return false
}

func isOutsideClassConstructor(node *ast.Node) bool {
	// a property initializer runs as part of construction, so `#dep = new Dep()` is fine - unless
	// the `new` sits in a function the initializer only stores, which runs whenever it is called.
	deferred := false
	for current := node.Parent; current != nil; current = current.Parent {
		switch current.Kind {
		case ast.KindArrowFunction, ast.KindFunctionExpression, ast.KindFunctionDeclaration:
			deferred = true
		case ast.KindConstructor:
			return false
		case ast.KindPropertyDeclaration:
			return deferred
		case ast.KindMethodDeclaration,
			ast.KindGetAccessor,
			ast.KindSetAccessor,
			ast.KindClassDeclaration,
			ast.KindClassExpression:
			return true
		}
	}
	return false
}

var NewOnlyInConstructorRule = rule.Rule{
	Name: "new-only-in-constructor",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		opts := utils.UnmarshalOptions[NewOnlyInConstructorOptions](options, "new-only-in-constructor")

		return rule.RuleListeners{
			ast.KindNewExpression: func(node *ast.Node) {
				name := constructedName(node)
				if name == "" || slices.Contains(opts.Allow, name) {
					return
				}
				if isEnclosingClassItself(node, name) {
					return
				}
				if isOutsideClassConstructor(node) {
					ctx.ReportNode(node, buildNewOutsideConstructorMessage(name))
				}
			},
		}
	},
}
