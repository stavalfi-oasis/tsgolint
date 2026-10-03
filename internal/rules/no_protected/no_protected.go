package no_protected

import (
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

// The one file allowed to declare protected members: ADisposable's `track` and
// `close` are the base-class surface every subclass inherits.
const disposableBaseFile = "shared/libs/src/a-disposable.ts"

func buildProtectedMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "protectedMember",
		Description: "`protected` is not allowed.",
		Help:        "Use a `#` prefix instead. A protected member is shared state between a class and everything that ever extends it.",
	}
}

var NoProtectedRule = rule.Rule{
	Name: "no-protected",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		if strings.HasSuffix(ctx.SourceFile.FileName(), disposableBaseFile) {
			return rule.RuleListeners{}
		}

		check := func(node *ast.Node) {
			if ast.HasSyntacticModifier(node, ast.ModifierFlagsProtected) {
				ctx.ReportNode(node, buildProtectedMessage())
			}
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
