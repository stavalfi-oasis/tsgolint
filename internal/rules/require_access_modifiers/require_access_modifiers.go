package require_access_modifiers

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

const accessibilityFlags = ast.ModifierFlagsPublic | ast.ModifierFlagsPrivate | ast.ModifierFlagsProtected

func buildMissingModifierMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "missingAccessModifier",
		Description: "Class members must declare an explicit `public` access modifier.",
		Help:        "Add `public`, or use a `#` prefix for a private member.",
	}
}

var RequireAccessModifiersRule = rule.Rule{
	Name: "require-access-modifiers",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		check := func(node *ast.Node) {
			if ast.HasSyntacticModifier(node, accessibilityFlags) {
				return
			}
			// `#field` is already private; an access modifier on it is a syntax error.
			if name := node.Name(); name != nil && ast.IsPrivateIdentifier(name) {
				return
			}
			ctx.ReportNodeWithFixes(node, buildMissingModifierMessage(), func() []rule.RuleFix {
				return []rule.RuleFix{rule.RuleFixInsertBefore(ctx.SourceFile, node, "public ")}
			})
		}

		return rule.RuleListeners{
			ast.KindConstructor:         check,
			ast.KindGetAccessor:         check,
			ast.KindMethodDeclaration:   check,
			ast.KindPropertyDeclaration: check,
			ast.KindSetAccessor:         check,
		}
	},
}
