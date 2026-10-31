package require_link_signal

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func buildRequireLinkSignalMessage(text string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "requireLinkSignal",
		Description: "`" + text + "` is the caller's signal only — it ignores this instance being closed.",
		Help:        "Use `super.linkSignal({ signal: " + text + " })` instead.",
	}
}

// Whether declaration is a parameter — or a binding inside one — of a method
// whose class inherits `linkSignal`. The base class that declares it extends
// nothing, so it is not checked against itself.
func isLinkableParameter(typeChecker *checker.Checker, declaration *ast.Node) bool {
	root := ast.GetRootDeclaration(declaration)
	if root.Kind != ast.KindParameter {
		return false
	}
	function := root.Parent
	switch function.Kind {
	case ast.KindMethodDeclaration, ast.KindConstructor, ast.KindGetAccessor, ast.KindSetAccessor:
	default:
		return false
	}
	classNode := function.Parent
	if classNode == nil || !ast.IsClassLike(classNode) || ast.GetExtendsHeritageClauseElement(classNode) == nil {
		return false
	}
	classSymbol := classNode.Symbol()
	if classSymbol == nil {
		return false
	}
	instanceType := checker.Checker_getDeclaredTypeOfSymbol(typeChecker, classSymbol)
	return instanceType != nil && checker.Checker_getPropertyOfType(typeChecker, instanceType, "linkSignal") != nil
}

func isAbortSignal(t *checker.Type) bool {
	return t != nil && utils.SomeUnionTypePart(t, func(part *checker.Type) bool {
		symbol := checker.Type_symbol(part)
		return symbol != nil && symbol.Name == "AbortSignal"
	})
}

// `super.linkSignal({ signal })` and `super.linkSignal({ signal: options.signal })`
// are the one place the raw parameter belongs.
func isLinkSignalArgument(node *ast.Node) bool {
	property := node.Parent
	switch {
	case property == nil:
		return false
	case property.Kind == ast.KindShorthandPropertyAssignment && property.Name() == node:
	case property.Kind == ast.KindPropertyAssignment && property.Initializer() == node:
	default:
		return false
	}
	object := property.Parent
	call := object.Parent
	if call == nil || !ast.IsCallExpression(call) {
		return false
	}
	arguments := call.AsCallExpression().Arguments
	if arguments == nil || len(arguments.Nodes) == 0 || arguments.Nodes[0] != object {
		return false
	}
	callee := call.AsCallExpression().Expression
	if !ast.IsPropertyAccessExpression(callee) {
		return false
	}
	access := callee.AsPropertyAccessExpression()
	return access.Expression.Kind == ast.KindSuperKeyword && access.Name().Text() == "linkSignal"
}

// The identifier a `a.b.c` chain starts from, or nil.
func rootIdentifier(node *ast.Node) *ast.Node {
	for ast.IsPropertyAccessExpression(node) {
		node = node.AsPropertyAccessExpression().Expression
	}
	if !ast.IsIdentifier(node) {
		return nil
	}
	return node
}

var RequireLinkSignalRule = rule.Rule{
	Name: "require-link-signal",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		check := func(node *ast.Node, symbol *ast.Symbol) {
			if symbol == nil || symbol.ValueDeclaration == nil || !isLinkableParameter(ctx.TypeChecker, symbol.ValueDeclaration) {
				return
			}
			if !isAbortSignal(ctx.TypeChecker.GetTypeAtLocation(node)) || isLinkSignalArgument(node) {
				return
			}
			textRange := utils.TrimNodeTextRange(ctx.SourceFile, node)
			ctx.ReportNode(node, buildRequireLinkSignalMessage(ctx.SourceFile.Text()[textRange.Pos():textRange.End()]))
		}

		return rule.RuleListeners{
			ast.KindIdentifier: func(node *ast.Node) {
				parent := node.Parent
				if parent != nil && parent.Kind == ast.KindShorthandPropertyAssignment && parent.Name() == node {
					check(node, checker.Checker_GetShorthandAssignmentValueSymbol(ctx.TypeChecker, parent))
					return
				}
				if ast.IsDeclarationName(node) {
					return
				}
				check(node, ctx.TypeChecker.GetSymbolAtLocation(node))
			},
			// `options.signal`, where `options` is the parameter.
			ast.KindPropertyAccessExpression: func(node *ast.Node) {
				root := rootIdentifier(node)
				if root == nil {
					return
				}
				check(node, ctx.TypeChecker.GetSymbolAtLocation(root))
			},
		}
	},
}
