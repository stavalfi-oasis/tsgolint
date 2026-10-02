// Package oasis holds helpers shared by the Oasis rules. It lives outside
// internal/utils on purpose: that file is upstream's, and keeping our helpers
// separate keeps `git rebase upstream/main` conflict-free.
package oasis

import (
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// DisposableBaseName is the class every Oasis service is expected to derive from.
const DisposableBaseName = "ADisposable"

// IsZodSchemaType reports whether the node's type is a zod schema. zod's runtime
// classes are all named ZodSomething (ZodObject, ZodString, ...), so the type
// identifies a schema no matter what the binding is called or how it was built.
func IsZodSchemaType(typeChecker *checker.Checker, node *ast.Node) bool {
	t := utils.GetConstrainedTypeAtLocation(typeChecker, node)
	return utils.SomeUnionTypePart(t, func(part *checker.Type) bool {
		symbol := checker.Type_symbol(part)
		return symbol != nil && strings.HasPrefix(symbol.Name, "Zod")
	})
}

// ExtendsDisposable walks the base-class chain, so a class extending a subclass
// of ADisposable is covered. The JS rules compared `superClass.name` to the
// literal string, so they only ever saw direct subclasses spelled that way.
// Types only ever *add* coverage here: the syntactic check below is what the JS
// rules did, and it still answers in files where the base class cannot be
// resolved — an untyped or `@ts-nocheck` file, or a synthesized snippet. The
// type walk adds the transitive and aliased cases on top.
func ExtendsDisposable(typeChecker *checker.Checker, class *ast.Node) bool {
	if extendsDisposableByName(class) {
		return true
	}

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
		if symbol := checker.Type_symbol(t); symbol != nil && symbol.Name == DisposableBaseName {
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

// The heritage clause spelled literally, which is all the JS rules ever checked.
func extendsDisposableByName(class *ast.Node) bool {
	clauses := class.ClassLikeData().HeritageClauses
	if clauses == nil {
		return false
	}
	for _, heritage := range clauses.Nodes {
		clause := heritage.AsHeritageClause()
		if clause.Token != ast.KindExtendsKeyword {
			continue
		}
		for _, typeNode := range clause.Types.Nodes {
			expression := typeNode.Expression()
			if expression != nil && ast.IsIdentifier(expression) &&
				expression.Text() == DisposableBaseName {
				return true
			}
		}
	}
	return false
}

// IsDisposableBase reports whether this class is ADisposable itself, which the
// rules exempt from their own requirements.
func IsDisposableBase(class *ast.Node) bool {
	name := class.Name()
	return name != nil && name.Text() == DisposableBaseName
}

// EnclosingInstanceClass returns the class a node sits in, or nil when the node
// is inside a static member — a static has no instance to track or dispose on.
func EnclosingInstanceClass(node *ast.Node) *ast.Node {
	for current := node.Parent; current != nil; current = current.Parent {
		if ast.IsClassLike(current) {
			return current
		}
		if ast.IsMethodDeclaration(current) || ast.IsPropertyDeclaration(current) {
			if ast.HasSyntacticModifier(current, ast.ModifierFlagsStatic) {
				return nil
			}
		}
	}
	return nil
}

// IsAsyncDisposeMember reports whether a class member is `[Symbol.asyncDispose]`.
func IsAsyncDisposeMember(member *ast.Node) bool {
	return isSymbolMember(member, "asyncDispose")
}

// IsDisposeMember reports whether a class member is either half of the disposal
// protocol — `[Symbol.dispose]` or `[Symbol.asyncDispose]`.
func IsDisposeMember(member *ast.Node) bool {
	return isSymbolMember(member, "dispose") || isSymbolMember(member, "asyncDispose")
}

// A method named by the computed key `[Symbol.<property>]`.
func isSymbolMember(member *ast.Node, property string) bool {
	if !ast.IsMethodDeclaration(member) {
		return false
	}
	name := member.Name()
	if name == nil || !ast.IsComputedPropertyName(name) {
		return false
	}
	expression := name.AsComputedPropertyName().Expression
	if !ast.IsPropertyAccessExpression(expression) {
		return false
	}
	access := expression.AsPropertyAccessExpression()
	return ast.IsIdentifier(access.Expression) &&
		access.Expression.Text() == "Symbol" &&
		access.Name().Text() == property
}
