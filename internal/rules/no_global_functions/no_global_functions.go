package no_global_functions

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildGlobalFunctionMessage(shape string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "globalFunction",
		Description: shape + " are banned.",
		Help:        "Use a class with static methods instead, so the file has one exported name.",
	}
}

var NoGlobalFunctionsRule = rule.Rule{
	Name: "no-global-functions",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		reportInitializer := func(initializer *ast.Node) {
			if initializer == nil {
				return
			}
			switch initializer.Kind {
			case ast.KindArrowFunction:
				ctx.ReportNode(initializer, buildGlobalFunctionMessage("Global arrow functions"))
			case ast.KindFunctionExpression:
				ctx.ReportNode(initializer, buildGlobalFunctionMessage("Global function expressions"))
			}
		}

		return rule.RuleListeners{
			ast.KindSourceFile: func(node *ast.Node) {
				for _, statement := range node.AsSourceFile().Statements.Nodes {
					switch statement.Kind {
					case ast.KindFunctionDeclaration:
						// `declare function f(): void` and an overload signature
						// introduce no runtime function; estree models both as
						// TSDeclareFunction, which the JS rule never looked at.
						if statement.Body() == nil {
							continue
						}
						ctx.ReportNode(statement, buildGlobalFunctionMessage("Global functions"))
					case ast.KindVariableStatement:
						declarationList := statement.AsVariableStatement().DeclarationList
						for _, declaration := range declarationList.AsVariableDeclarationList().Declarations.Nodes {
							reportInitializer(declaration.Initializer())
						}
					case ast.KindExportAssignment:
						reportInitializer(statement.AsExportAssignment().Expression)
					}
				}
			},
		}
	},
}
