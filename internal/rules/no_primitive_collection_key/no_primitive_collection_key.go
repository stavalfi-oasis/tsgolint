package no_primitive_collection_key

import (
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

type NoPrimitiveCollectionKeyOptions struct {
	// Added to the built-in container list, never replacing it.
	Containers []string `json:"containers"`
	// Added to the built-in denylist, never replacing it.
	MeaninglessNames []string `json:"meaninglessNames"`
}

// Every one of these is keyed by its first type argument — for the set types
// that single argument is the key.
var defaultContainers = []string{
	"Map", "WeakMap", "ReadonlyMap",
	"Set", "WeakSet", "ReadonlySet",
	"Record",
}

// Only `Map`, `Set`, `WeakMap` and `WeakSet` have constructors; the rest are
// type-only, so `new` never reaches them.
var constructibleContainers = []string{"Map", "WeakMap", "Set", "WeakSet"}

// Names that rename a primitive without saying anything about it. Compared
// case-insensitively, so `Str`, `str` and `STR` are all caught.
var defaultMeaninglessNames = []string{
	"str", "string", "num", "number", "int", "integer", "float",
	"bool", "boolean", "key", "keys", "val", "value", "values",
	"data", "item", "thing", "tmp", "temp", "foo", "bar", "baz",
	"x", "y", "z", "t", "s", "n", "b",
}

func buildPrimitiveKeyMessage(container string, keyword string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "primitiveKey",
		Description: "`" + container + "` is keyed by a bare `" + keyword + "`, which says nothing about what the key holds.",
		Help:        "Alias the key to a named type that says what it represents — `type TenantId = " + keyword + "` — and key on `TenantId`.",
	}
}

func buildMeaninglessAliasMessage(container string, name string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "meaninglessAlias",
		Description: "`" + name + "` renames a primitive without naming it, so `" + container + "` still has an anonymous key.",
		Help:        "Rename the alias to what the key represents, such as `TenantId` or `FindingSlug`.",
	}
}

func buildMissingTypeArgumentsMessage(container string) rule.RuleMessage {
	// `Set` and `WeakSet` take the key alone; `Map` and `WeakMap` take a value too.
	example := "<TenantId, Session>"
	if strings.HasSuffix(container, "Set") {
		example = "<TenantId>"
	}
	return rule.RuleMessage{
		Id:          "missingTypeArguments",
		Description: "`new " + container + "()` leaves the key type to inference, so nothing in the source names it.",
		Help:        "Write the type arguments explicitly with a named alias as the key — `new " + container + example + "()`.",
	}
}

func primitiveKeyword(node *ast.Node) string {
	switch node.Kind {
	case ast.KindStringKeyword:
		return "string"
	case ast.KindNumberKeyword:
		return "number"
	case ast.KindBooleanKeyword:
		return "boolean"
	}
	return ""
}

func unwrapParenthesizedType(node *ast.Node) *ast.Node {
	for node != nil && node.Kind == ast.KindParenthesizedType {
		node = node.AsParenthesizedTypeNode().Type
	}
	return node
}

// `ns.TenantId` names the key just as well as `TenantId` does.
func rightmostIdentifier(name *ast.Node) *ast.Node {
	if name != nil && name.Kind == ast.KindQualifiedName {
		return name.AsQualifiedName().Right
	}
	return name
}

func containerName(expression *ast.Node) string {
	if expression == nil {
		return ""
	}
	if ast.IsIdentifier(expression) {
		return expression.Text()
	}
	if ast.IsPropertyAccessExpression(expression) {
		return expression.AsPropertyAccessExpression().Name().Text()
	}
	return ""
}

// A `new Map()` inside an annotated declaration already has its key named by
// that annotation, which this rule checks on its own. Only a declaration with
// nothing written on it leaves the key unnamed.
func inferenceLeavesKeyUnnamed(node *ast.Node) bool {
	parent := node.Parent
	if parent == nil {
		return false
	}
	switch parent.Kind {
	case ast.KindVariableDeclaration:
		return parent.AsVariableDeclaration().Type == nil
	case ast.KindPropertyDeclaration:
		return parent.AsPropertyDeclaration().Type == nil
	}
	return false
}

var NoPrimitiveCollectionKeyRule = rule.Rule{
	Name: "no-primitive-collection-key",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		opts := utils.UnmarshalOptions[NoPrimitiveCollectionKeyOptions](options, "no-primitive-collection-key")

		containers := slices.Concat(defaultContainers, opts.Containers)

		meaningless := make([]string, 0, len(defaultMeaninglessNames)+len(opts.MeaninglessNames))
		meaningless = append(meaningless, defaultMeaninglessNames...)
		for _, name := range opts.MeaninglessNames {
			meaningless = append(meaningless, strings.ToLower(name))
		}

		// A type parameter is the caller's to name; the rule has nothing to say
		// about `Map<K, V>` inside a generic helper.
		isTypeParameter := func(identifier *ast.Node) bool {
			symbol := ctx.TypeChecker.GetSymbolAtLocation(identifier)
			return symbol != nil && symbol.Flags&ast.SymbolFlagsTypeParameter != 0
		}

		var checkKey func(container string, key *ast.Node)
		checkKey = func(container string, key *ast.Node) {
			key = unwrapParenthesizedType(key)
			if key == nil {
				return
			}

			if keyword := primitiveKeyword(key); keyword != "" {
				ctx.ReportNode(key, buildPrimitiveKeyMessage(container, keyword))
				return
			}

			switch key.Kind {
			// `Map<'read' | 'write', X>` already names every key it accepts.
			case ast.KindLiteralType:
				return
			case ast.KindUnionType:
				for _, constituent := range key.AsUnionTypeNode().Types.Nodes {
					checkKey(container, constituent)
				}
			// An inline brand — `string & { __brand: 'TenantId' }` — still leaves
			// the key unnamed at the use site.
			case ast.KindIntersectionType:
				for _, constituent := range key.AsIntersectionTypeNode().Types.Nodes {
					checkKey(container, constituent)
				}
			case ast.KindTypeReference:
				identifier := rightmostIdentifier(key.AsTypeReferenceNode().TypeName)
				if identifier == nil || !ast.IsIdentifier(identifier) {
					return
				}
				if isTypeParameter(identifier) {
					return
				}
				name := identifier.Text()
				if slices.Contains(meaningless, strings.ToLower(name)) {
					ctx.ReportNode(key, buildMeaninglessAliasMessage(container, name))
				}
			}
		}

		checkTypeArguments := func(name string, typeArguments *ast.NodeList) {
			if !slices.Contains(containers, name) {
				return
			}
			if typeArguments == nil || len(typeArguments.Nodes) == 0 {
				return
			}
			checkKey(name, typeArguments.Nodes[0])
		}

		return rule.RuleListeners{
			ast.KindTypeReference: func(node *ast.Node) {
				reference := node.AsTypeReferenceNode()
				identifier := rightmostIdentifier(reference.TypeName)
				if identifier == nil || !ast.IsIdentifier(identifier) {
					return
				}
				checkTypeArguments(identifier.Text(), reference.TypeArguments)
			},
			// `class Sessions extends Map<string, Session> {}`
			ast.KindExpressionWithTypeArguments: func(node *ast.Node) {
				expression := node.AsExpressionWithTypeArguments()
				checkTypeArguments(containerName(expression.Expression), expression.TypeArguments)
			},
			ast.KindNewExpression: func(node *ast.Node) {
				expression := node.AsNewExpression()
				name := containerName(expression.Expression)
				if !slices.Contains(constructibleContainers, name) {
					return
				}
				if expression.TypeArguments != nil && len(expression.TypeArguments.Nodes) > 0 {
					checkKey(name, expression.TypeArguments.Nodes[0])
					return
				}
				if inferenceLeavesKeyUnnamed(node) {
					ctx.ReportNode(expression.Expression, buildMissingTypeArgumentsMessage(name))
				}
			},
		}
	},
}
