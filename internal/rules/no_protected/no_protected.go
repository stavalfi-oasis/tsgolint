package no_protected

import (
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

// The files allowed to declare protected members: these are base-class surfaces
// every subclass inherits — ADisposable's `track` and `close`, and
// TypedClient's `parsedJson`.
var baseClassFiles = []string{
	"shared/libs/src/a-disposable.ts",
	"shared/libs/src/typed-client.ts",
}

func isBaseClassFile(fileName string) bool {
	for _, baseClassFile := range baseClassFiles {
		if strings.HasSuffix(fileName, baseClassFile) {
			return true
		}
	}
	return false
}

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
		if isBaseClassFile(ctx.SourceFile.FileName()) {
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
