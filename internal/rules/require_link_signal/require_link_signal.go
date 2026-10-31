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

func buildNoLinkSignalInPrivateMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "noLinkSignalInPrivate",
		Description: "A private method already gets a linked signal from the public method that calls it.",
		Help:        "Pass the signal through as-is instead of calling `super.linkSignal` again.",
	}
}

// A private method is only reached from this class's own methods, which
// already hand it a linked signal — linking again would wrap it twice.
func isPrivateMethod(function *ast.Node) bool {
	if ast.HasSyntacticModifier(function, ast.ModifierFlagsPrivate) {
		return true
	}
	name := function.Name()
	return name != nil && ast.IsPrivateIdentifier(name)
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
	// `linkSignal` is an instance member; a static method has no `super` to reach it.
	if ast.IsStatic(function) {
		return false
	}
	if isPrivateMethod(function) {
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
	if call == nil || !isSuperLinkSignalCall(call) {
		return false
	}
	return call.AsCallExpression().Arguments.Nodes[0] == object
}

// `super.linkSignal(<at least one argument>)`.
func isSuperLinkSignalCall(node *ast.Node) bool {
	if !ast.IsCallExpression(node) {
		return false
	}
	arguments := node.AsCallExpression().Arguments
	if arguments == nil || len(arguments.Nodes) == 0 {
		return false
	}
	callee := node.AsCallExpression().Expression
	if !ast.IsPropertyAccessExpression(callee) {
		return false
	}
	access := callee.AsPropertyAccessExpression()
	return access.Expression.Kind == ast.KindSuperKeyword && access.Name().Text() == "linkSignal"
}

// The method `super` binds to at node, or nil inside a nested `function`,
// where `super` is not valid.
func enclosingMethod(node *ast.Node) *ast.Node {
	for current := node.Parent; current != nil; current = current.Parent {
		switch current.Kind {
		case ast.KindMethodDeclaration, ast.KindConstructor, ast.KindGetAccessor, ast.KindSetAccessor:
			return current
		case ast.KindFunctionExpression, ast.KindFunctionDeclaration:
			return nil
		}
	}
	return nil
}

// `super` is only valid up to the nearest non-arrow function, so the fix stops
// being safe inside a nested `function` of the method.
func canReachSuper(node *ast.Node) bool {
	return enclosingMethod(node) != nil
}

// The signal expression inside `{ signal }` / `{ signal: <expr> }`, or nil when
// the argument has any other shape.
func linkedSignalExpression(argument *ast.Node) *ast.Node {
	if !ast.IsObjectLiteralExpression(argument) {
		return nil
	}
	properties := argument.AsObjectLiteralExpression().Properties
	if properties == nil || len(properties.Nodes) != 1 {
		return nil
	}
	property := properties.Nodes[0]
	switch property.Kind {
	case ast.KindShorthandPropertyAssignment:
		if property.Name().Text() == "signal" {
			return property.Name()
		}
	case ast.KindPropertyAssignment:
		if name := property.Name(); ast.IsIdentifier(name) && name.Text() == "signal" {
			return property.Initializer()
		}
	}
	return nil
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
			text := ctx.SourceFile.Text()[textRange.Pos():textRange.End()]
			message := buildRequireLinkSignalMessage(text)
			if !canReachSuper(node) {
				ctx.ReportNode(node, message)
				return
			}
			ctx.ReportNodeWithFixes(node, message, func() []rule.RuleFix {
				linked := "super.linkSignal({ signal: " + text + " })"
				if text == "signal" {
					linked = "super.linkSignal({ signal })"
				}
				if parent := node.Parent; parent.Kind == ast.KindShorthandPropertyAssignment {
					return []rule.RuleFix{rule.RuleFixReplace(ctx.SourceFile, parent, text+": "+linked)}
				}
				return []rule.RuleFix{rule.RuleFixReplaceRange(textRange, linked)}
			})
		}

		return rule.RuleListeners{
			// `super.linkSignal(...)` inside a private method wraps an already-linked signal.
			ast.KindCallExpression: func(node *ast.Node) {
				if !isSuperLinkSignalCall(node) {
					return
				}
				method := enclosingMethod(node)
				if method == nil || !isPrivateMethod(method) {
					return
				}
				signal := linkedSignalExpression(node.AsCallExpression().Arguments.Nodes[0])
				if signal == nil {
					ctx.ReportNode(node, buildNoLinkSignalInPrivateMessage())
					return
				}
				ctx.ReportNodeWithFixes(node, buildNoLinkSignalInPrivateMessage(), func() []rule.RuleFix {
					signalRange := utils.TrimNodeTextRange(ctx.SourceFile, signal)
					text := ctx.SourceFile.Text()[signalRange.Pos():signalRange.End()]
					return []rule.RuleFix{rule.RuleFixReplaceRange(utils.TrimNodeTextRange(ctx.SourceFile, node), text)}
				})
			},
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
