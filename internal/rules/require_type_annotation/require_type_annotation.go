package require_type_annotation

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildMissingTypeAnnotationMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "missingTypeAnnotation",
		Description: "A declaration with no initializer must carry a type annotation.",
		Help:        "Write the type here, or give the declaration an initializer on the same line.",
	}
}

var RequireTypeAnnotationRule = rule.Rule{
	Name: "require-type-annotation",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindPropertyDeclaration: func(node *ast.Node) {
				property := node.AsPropertyDeclaration()
				// An initializer on the same line already states the type at the
				// declaration; only a bare `name;` leaves the reader nothing to read.
				if property.Type != nil || property.Initializer != nil {
					return
				}
				ctx.ReportNode(node, buildMissingTypeAnnotationMessage())
			},
			ast.KindVariableDeclaration: func(node *ast.Node) {
				declaration := node.AsVariableDeclaration()
				if declaration.Type != nil || declaration.Initializer != nil {
					return
				}
				// `catch (error)` and `for (const x of xs)` have no initializer to
				// annotate against, and the binding is not a declaration site a
				// reader looks at for the type.
				parent := node.Parent
				if parent != nil && parent.Kind == ast.KindCatchClause {
					return
				}
				if parent != nil && parent.Kind == ast.KindVariableDeclarationList {
					grandparent := parent.Parent
					if grandparent != nil &&
						(grandparent.Kind == ast.KindForInStatement ||
							grandparent.Kind == ast.KindForOfStatement) {
						return
					}
				}
				ctx.ReportNode(node, buildMissingTypeAnnotationMessage())
			},
		}
	},
}
