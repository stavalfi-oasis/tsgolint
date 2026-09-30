package no_import_side_effects

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildRunsAtImportMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "runsAtImport",
		Description: "This runs when the module is imported.",
		Help:        "Move it into a class static method the entrypoint calls.",
	}
}

func buildSideEffectImportMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "sideEffectImport",
		Description: "Side-effect-only imports are banned.",
		Help:        "Import a value and call it explicitly.",
	}
}

func buildStaticPropertyMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "staticProperty",
		Description: "Static class properties are initialized when the module is imported.",
		Help:        "Return the value from a class static method instead.",
	}
}

// Top-level forms that only declare. Everything else at module scope runs.
var declarativeKinds = []ast.Kind{
	ast.KindClassDeclaration,
	ast.KindEmptyStatement,
	ast.KindEnumDeclaration,
	ast.KindExportDeclaration,
	ast.KindFunctionDeclaration,
	ast.KindImportDeclaration,
	ast.KindImportEqualsDeclaration,
	ast.KindInterfaceDeclaration,
	ast.KindModuleDeclaration,
	ast.KindTypeAliasDeclaration,
	ast.KindVariableStatement,
}

// `promisify(fn)` builds a function, it does not call one.
var allowedCallees = []string{"promisify"}

// A test file's `describe(...)` / `it(...)` register cases; they are the file's
// whole point and cannot move into a method.
var allowedTopLevelCalls = []string{"describe", "it"}

func rootCalleeName(callee *ast.Node) string {
	if ast.IsIdentifier(callee) {
		return callee.Text()
	}
	if ast.IsPropertyAccessExpression(callee) {
		if object := callee.AsPropertyAccessExpression().Expression; ast.IsIdentifier(object) {
			return object.Text()
		}
	}
	return ""
}

func isAllowedTopLevelCall(statement *ast.Node) bool {
	if !ast.IsExpressionStatement(statement) {
		return false
	}
	expression := ast.SkipParentheses(statement.AsExpressionStatement().Expression)
	if !ast.IsCallExpression(expression) {
		return false
	}
	return slices.Contains(allowedTopLevelCalls, rootCalleeName(expression.AsCallExpression().Expression))
}

var NoImportSideEffectsRule = rule.Rule{
	Name: "no-import-side-effects",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		// Spans of code that the module body evaluates on import...
		evaluated := []core.TextRange{}
		// ...minus the spans that only run when something calls them...
		functions := []core.TextRange{}
		// ...and one report per outermost offender, not per nested expression.
		reported := []core.TextRange{}

		contains := func(outer core.TextRange, inner *ast.Node) bool {
			return inner.Pos() >= outer.Pos() && inner.End() <= outer.End()
		}

		rememberFunction := func(node *ast.Node) {
			functions = append(functions, node.Loc)
		}

		reportIfImportTime := func(node *ast.Node) {
			if !slices.ContainsFunc(evaluated, func(outer core.TextRange) bool { return contains(outer, node) }) {
				return
			}
			if slices.ContainsFunc(functions, func(outer core.TextRange) bool { return contains(outer, node) }) {
				return
			}
			if slices.ContainsFunc(reported, func(outer core.TextRange) bool { return contains(outer, node) }) {
				return
			}
			reported = append(reported, node.Loc)
			ctx.ReportNode(node, buildRunsAtImportMessage())
		}

		return rule.RuleListeners{
			ast.KindSourceFile: func(node *ast.Node) {
				for _, statement := range node.AsSourceFile().Statements.Nodes {
					if isAllowedTopLevelCall(statement) {
						continue
					}
					// `export default <expression>` evaluates the expression.
					if statement.Kind == ast.KindExportAssignment {
						evaluated = append(evaluated, statement.AsExportAssignment().Expression.Loc)
						continue
					}
					if !slices.Contains(declarativeKinds, statement.Kind) {
						ctx.ReportNode(statement, buildRunsAtImportMessage())
						continue
					}
					if statement.Kind != ast.KindVariableStatement {
						continue
					}
					declarationList := statement.AsVariableStatement().DeclarationList
					for _, declaration := range declarationList.AsVariableDeclarationList().Declarations.Nodes {
						if initializer := declaration.Initializer(); initializer != nil {
							evaluated = append(evaluated, initializer.Loc)
						}
					}
				}
			},

			ast.KindArrowFunction:       rememberFunction,
			ast.KindFunctionDeclaration: rememberFunction,
			ast.KindFunctionExpression:  rememberFunction,

			ast.KindAwaitExpression:          reportIfImportTime,
			ast.KindImportKeyword:            reportIfImportTime,
			ast.KindNewExpression:            reportIfImportTime,
			ast.KindTaggedTemplateExpression: reportIfImportTime,

			ast.KindCallExpression: func(node *ast.Node) {
				callee := node.AsCallExpression().Expression
				if ast.IsIdentifier(callee) && slices.Contains(allowedCallees, callee.Text()) {
					return
				}
				reportIfImportTime(node)
			},

			ast.KindImportDeclaration: func(node *ast.Node) {
				if node.AsImportDeclaration().ImportClause == nil {
					ctx.ReportNode(node, buildSideEffectImportMessage())
				}
			},

			ast.KindPropertyDeclaration: func(node *ast.Node) {
				if ast.HasSyntacticModifier(node, ast.ModifierFlagsStatic) {
					ctx.ReportNode(node, buildStaticPropertyMessage())
				}
			},
		}
	},
}
