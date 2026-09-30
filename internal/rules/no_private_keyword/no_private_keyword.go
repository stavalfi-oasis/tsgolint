package no_private_keyword

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildPrivateKeywordMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "privateKeyword",
		Description: "`private` keyword is not allowed.",
		Help:        "Use a `#` prefix instead, so the field is private at runtime and not only to the type checker.",
	}
}

var NoPrivateKeywordRule = rule.Rule{
	Name: "no-private-keyword",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		// The constructor is exempt: `private` there is a parameter property, and
		// there is no `#` spelling for one.
		check := func(node *ast.Node) {
			if ast.HasSyntacticModifier(node, ast.ModifierFlagsPrivate) {
				ctx.ReportNode(node, buildPrivateKeywordMessage())
			}
		}

		return rule.RuleListeners{
			ast.KindGetAccessor:        check,
			ast.KindMethodDeclaration:  check,
			ast.KindPropertyDeclaration: check,
			ast.KindSetAccessor:        check,
		}
	},
}
