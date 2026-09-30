package no_single_use_interface

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildSingleUseMessage(name string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "singleUseInterface",
		Description: "Interface '" + name + "' is used only once.",
		Help:        "Inline its members at the single usage instead of introducing a single-use interface.",
	}
}

var NoSingleUseInterfaceRule = rule.Rule{
	Name: "no-single-use-interface",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		type declared struct {
			node *ast.Node
			name string
		}

		local := map[*ast.Symbol]*declared{}
		uses := map[*ast.Symbol]int{}

		return rule.RuleListeners{
			ast.KindInterfaceDeclaration: func(node *ast.Node) {
				// An exported interface is part of the package's surface; its
				// other uses live in files this run may not even see.
				if ast.HasSyntacticModifier(node, ast.ModifierFlagsExport) {
					return
				}
				name := node.Name()
				if name == nil {
					return
				}
				symbol := ctx.TypeChecker.GetSymbolAtLocation(name)
				if symbol == nil {
					return
				}
				local[symbol] = &declared{node: name, name: name.Text()}
			},

			// Counting resolved symbols rather than matching identifier text is
			// what makes this correct: the JS rule shelled out to `git ls-files`
			// and regex-scanned every file's import statements, so a type used
			// under an alias counted as unused and a same-named type in another
			// file counted as a use.
			ast.KindTypeReference: func(node *ast.Node) {
				name := node.AsTypeReferenceNode().TypeName
				for ast.IsQualifiedName(name) {
					name = name.AsQualifiedName().Right
				}
				symbol := ctx.TypeChecker.GetSymbolAtLocation(name)
				if symbol == nil {
					return
				}
				uses[symbol]++
			},

			rule.ListenerOnExit(ast.KindSourceFile): func(node *ast.Node) {
				for symbol, decl := range local {
					if uses[symbol] == 1 {
						ctx.ReportNode(decl.node, buildSingleUseMessage(decl.name))
					}
				}
			},
		}
	},
}
