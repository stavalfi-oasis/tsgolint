package no_single_use_const

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildSingleUseMessage(name string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "singleUseConst",
		Description: "'" + name + "' is a constant used only once.",
		Help:        "Inline its value at the single usage instead of introducing a single-use constant.",
	}
}

// Only values that read the same inlined: a string, a number, or a template.
func isInlinableInitializer(initializer *ast.Node) bool {
	if initializer == nil {
		return false
	}
	switch initializer.Kind {
	case ast.KindStringLiteral,
		ast.KindNumericLiteral,
		ast.KindNoSubstitutionTemplateLiteral,
		ast.KindTemplateExpression:
		return true
	}
	return false
}

var NoSingleUseConstRule = rule.Rule{
	Name: "no-single-use-const",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		type declared struct {
			name *ast.Node
		}

		candidates := map[*ast.Symbol]*declared{}
		reads := map[*ast.Symbol]int{}

		return rule.RuleListeners{
			ast.KindVariableStatement: func(node *ast.Node) {
				// An exported const is part of the module's surface; its other
				// uses live in files this run may not even see.
				if ast.HasSyntacticModifier(node, ast.ModifierFlagsExport) {
					return
				}
				declarationList := node.AsVariableStatement().DeclarationList
				if declarationList.Flags&ast.NodeFlagsConst == 0 {
					return
				}
				for _, declaration := range declarationList.AsVariableDeclarationList().Declarations.Nodes {
					name := declaration.Name()
					if name == nil || !ast.IsIdentifier(name) {
						continue
					}
					if !isInlinableInitializer(declaration.Initializer()) {
						continue
					}
					if symbol := ctx.TypeChecker.GetSymbolAtLocation(name); symbol != nil {
						candidates[symbol] = &declared{name: name}
					}
				}
			},

			ast.KindIdentifier: func(node *ast.Node) {
				// The binding itself is a declaration, not a read.
				if parent := node.Parent; parent != nil && ast.IsVariableDeclaration(parent) &&
					parent.Name() == node {
					return
				}
				if symbol := ctx.TypeChecker.GetSymbolAtLocation(node); symbol != nil {
					reads[symbol]++
				}
			},

			rule.ListenerOnExit(ast.KindSourceFile): func(node *ast.Node) {
				for symbol, candidate := range candidates {
					if reads[symbol] == 1 {
						ctx.ReportNode(candidate.name, buildSingleUseMessage(candidate.name.Text()))
					}
				}
			},
		}
	},
}
