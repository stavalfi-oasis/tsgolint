package no_banned_types_in_exports

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

type NoBannedTypesInExportsOptions struct {
	Types []string `json:"types"`
}

func buildBannedTypeMessage(name string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "bannedTypeInExport",
		Description: "'" + name + "' must not be used in an exported type.",
		Help:        "Spell the exported type out explicitly, or apply '" + name + "' at the use site instead.",
	}
}

// The nearest type alias or interface that owns this reference, or nil when the
// reference sits in a function, class or variable rather than a type declaration.
func enclosingTypeDeclaration(node *ast.Node) *ast.Node {
	for current := node.Parent; current != nil; current = current.Parent {
		switch current.Kind {
		case ast.KindTypeAliasDeclaration, ast.KindInterfaceDeclaration:
			return current
		case ast.KindSourceFile, ast.KindBlock, ast.KindModuleBlock,
			ast.KindFunctionDeclaration, ast.KindClassDeclaration, ast.KindVariableStatement:
			return nil
		}
	}
	return nil
}

var NoBannedTypesInExportsRule = rule.Rule{
	Name: "no-banned-types-in-exports",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		opts := utils.UnmarshalOptions[NoBannedTypesInExportsOptions](options, "no-banned-types-in-exports")

		// Resolving through the import alias is what catches
		// `import type { DeepReadonly as DR }` — the written name is not the banned one.
		resolvedName := func(name *ast.Node) string {
			symbol := ctx.TypeChecker.GetSymbolAtLocation(name)
			if symbol == nil {
				return name.Text()
			}
			if utils.IsSymbolFlagSet(symbol, ast.SymbolFlagsAlias) {
				symbol = ctx.TypeChecker.GetAliasedSymbol(symbol)
			}
			if symbol == nil || symbol.Name == "" {
				return name.Text()
			}
			return symbol.Name
		}

		return rule.RuleListeners{
			ast.KindTypeReference: func(node *ast.Node) {
				typeName := node.AsTypeReferenceNode().TypeName
				for ast.IsQualifiedName(typeName) {
					typeName = typeName.AsQualifiedName().Right
				}
				name := resolvedName(typeName)
				if !slices.Contains(opts.Types, name) {
					return
				}
				declaration := enclosingTypeDeclaration(node)
				if declaration == nil || !ast.HasSyntacticModifier(declaration, ast.ModifierFlagsExport) {
					return
				}
				// The banned type's own definition may refer to itself.
				if declarationName := declaration.Name(); declarationName != nil && declarationName.Text() == name {
					return
				}
				ctx.ReportNode(node, buildBannedTypeMessage(name))
			},
		}
	},
}
