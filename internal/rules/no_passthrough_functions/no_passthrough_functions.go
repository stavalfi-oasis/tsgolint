package no_passthrough_functions

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildPassthroughMessage(name string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "passthroughFunction",
		Description: "This function only forwards its arguments to '" + name + "'.",
		Help:        "Call '" + name + "' directly at the call sites and delete this wrapper.",
	}
}

func unwrap(node *ast.Node) *ast.Node {
	current := ast.SkipParentheses(node)
	for {
		switch {
		case ast.IsAwaitExpression(current):
			current = ast.SkipParentheses(current.AsAwaitExpression().Expression)
		case current.Kind == ast.KindNonNullExpression:
			current = ast.SkipParentheses(current.AsNonNullExpression().Expression)
		default:
			return current
		}
	}
}

// The single expression a function body evaluates to, or nil when the body does
// more than one thing.
func soleExpression(body *ast.Node) *ast.Node {
	if body == nil {
		return nil
	}
	if !ast.IsBlock(body) {
		return body
	}
	statements := body.AsBlock().Statements.Nodes
	if len(statements) != 1 {
		return nil
	}
	switch statement := statements[0]; {
	case ast.IsExpressionStatement(statement):
		return statement.AsExpressionStatement().Expression
	case ast.IsReturnStatement(statement):
		return statement.AsReturnStatement().Expression
	default:
		return nil
	}
}

func parameterNames(parameters []*ast.Node) map[string]bool {
	names := map[string]bool{}
	for _, parameter := range parameters {
		name := parameter.Name()
		if name != nil && ast.IsIdentifier(name) {
			names[name.Text()] = true
		}
	}
	return names
}

// Every argument must be one of this function's own parameters, forwarded
// as-is. Anything computed means the wrapper does real work.
func forwardsOnlyParameters(call *ast.CallExpression, parameters map[string]bool) bool {
	if call.Arguments == nil {
		return len(parameters) == 0
	}
	if len(call.Arguments.Nodes) == 0 {
		return false
	}
	for _, argument := range call.Arguments.Nodes {
		value := argument
		if value.Kind == ast.KindSpreadElement {
			value = value.AsSpreadElement().Expression
		}
		value = unwrap(value)
		if !ast.IsIdentifier(value) || !parameters[value.Text()] {
			return false
		}
	}
	return true
}

// Resolves the callee to its declaration and requires it to live in this file.
// The JS rule kept name sets of local functions and imports to approximate
// this, so a local shadowing an import — or any indirection — fooled it.
func localTargetName(typeChecker *checker.Checker, sourceFile *ast.SourceFile, call *ast.CallExpression) string {
	callee := unwrap(call.Expression)

	var nameNode *ast.Node
	switch {
	case ast.IsIdentifier(callee):
		nameNode = callee
	case ast.IsPropertyAccessExpression(callee):
		access := callee.AsPropertyAccessExpression()
		if access.Expression.Kind != ast.KindThisKeyword {
			return ""
		}
		nameNode = access.Name()
	default:
		return ""
	}

	symbol := typeChecker.GetSymbolAtLocation(nameNode)
	if symbol == nil || len(symbol.Declarations) == 0 {
		return ""
	}
	for _, declaration := range symbol.Declarations {
		if ast.GetSourceFileOfNode(declaration) != sourceFile {
			return ""
		}
		// An import specifier lives in this file but the function does not —
		// wrapping an import is the local seam, which is the point.
		switch declaration.Kind {
		case ast.KindImportSpecifier, ast.KindImportClause, ast.KindNamespaceImport,
			ast.KindImportEqualsDeclaration:
			return ""
		}
	}
	return symbol.Name
}

var NoPassthroughFunctionsRule = rule.Rule{
	Name: "no-passthrough-functions",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		check := func(node *ast.Node) {
			body := node.Body()
			expression := soleExpression(body)
			if expression == nil {
				return
			}

			call := unwrap(expression)
			if !ast.IsCallExpression(call) {
				return
			}

			parameters := parameterNames(node.Parameters())
			if len(parameters) == 0 {
				return
			}
			if !forwardsOnlyParameters(call.AsCallExpression(), parameters) {
				return
			}

			target := localTargetName(ctx.TypeChecker, ctx.SourceFile, call.AsCallExpression())
			if target == "" {
				return
			}

			reported := node.Name()
			if reported == nil {
				reported = node
			}
			ctx.ReportNode(reported, buildPassthroughMessage(target))
		}

		return rule.RuleListeners{
			ast.KindFunctionDeclaration: check,
			ast.KindFunctionExpression:  check,
			ast.KindArrowFunction:       check,
			ast.KindMethodDeclaration:   check,
		}
	},
}
