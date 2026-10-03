package prefer_super_over_this

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildPreferSuperMessage(name string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "preferSuper",
		Description: fmt.Sprintf("`%s` comes from a base class — reach it through `super`, not `this`.", name),
		Help:        fmt.Sprintf("Write `super.%s` so the reader can see the member is inherited.", name),
	}
}

// The class element a `this` inside `node` belongs to, or nil when `this` is
// not the instance of a class. GetThisContainer with includeArrowFunctions
// false walks past arrow functions (which keep both `this` and `super`) and
// stops at the first `function` (which rebinds `this` and makes `super` a
// syntax error), so a non-class-element container is the bail-out signal.
func containingClassElement(node *ast.Node) *ast.Node {
	container := ast.GetThisContainer(node, false, false)
	if container == nil || container.Parent == nil || !ast.IsClassLike(container.Parent) {
		return nil
	}
	switch container.Kind {
	case ast.KindMethodDeclaration, ast.KindConstructor, ast.KindGetAccessor,
		ast.KindSetAccessor, ast.KindPropertyDeclaration:
		return container
	default:
		return nil
	}
}

// Only a prototype member is reachable through `super`. `super.field` on an
// instance field is always undefined, and an abstract method has no base
// implementation to dispatch to, so both would turn a working call into a
// broken one.
func isSuperReachable(declaration *ast.Node) bool {
	switch declaration.Kind {
	case ast.KindMethodDeclaration, ast.KindGetAccessor, ast.KindSetAccessor:
	default:
		return false
	}
	if declaration.Parent == nil || !ast.IsClassLike(declaration.Parent) {
		return false
	}
	return !ast.IsStatic(declaration) &&
		!ast.HasSyntacticModifier(declaration, ast.ModifierFlagsAbstract)
}

var PreferSuperOverThisRule = rule.Rule{
	Name: "prefer-super-over-this",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		// `this.m` is only interchangeable with `super.m` where the result is
		// consumed immediately: as the callee of a call, or as an accessor read
		// or write. Handing `this.m` around as a value and handing `super.m`
		// around as a value are different programs.
		isImmediatelyConsumed := func(node *ast.Node, declaration *ast.Node) bool {
			if ast.IsAccessor(declaration) {
				return true
			}
			parent := node.Parent
			return parent != nil && ast.IsCallExpression(parent) &&
				parent.AsCallExpression().Expression == node
		}

		return rule.RuleListeners{
			ast.KindPropertyAccessExpression: func(node *ast.Node) {
				access := node.AsPropertyAccessExpression()
				if access.Expression.Kind != ast.KindThisKeyword {
					return
				}
				// `super?.m` parses, but an optional chain off `super` is not
				// worth rewriting into.
				if access.QuestionDotToken != nil {
					return
				}
				name := access.Name()
				// `this.#x` is never inherited — a private name is scoped to the
				// class that declares it.
				if name == nil || name.Kind != ast.KindIdentifier {
					return
				}

				element := containingClassElement(node)
				if element == nil || ast.IsStatic(element) {
					return
				}
				classNode := element.Parent
				if ast.GetExtendsHeritageClauseElement(classNode) == nil {
					return
				}

				symbol := ctx.TypeChecker.GetSymbolAtLocation(name)
				if symbol == nil || len(symbol.Declarations) == 0 {
					return
				}
				for _, declaration := range symbol.Declarations {
					// Declared here too: `this` is the override and is correct.
					if declaration.Parent == classNode || !isSuperReachable(declaration) {
						return
					}
				}
				if !isImmediatelyConsumed(node, symbol.Declarations[0]) {
					return
				}

				ctx.ReportNodeWithFixes(access.Expression, buildPreferSuperMessage(name.Text()), func() []rule.RuleFix {
					return []rule.RuleFix{rule.RuleFixReplace(ctx.SourceFile, access.Expression, "super")}
				})
			},
		}
	},
}
