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
func localTargetSymbol(typeChecker *checker.Checker, sourceFile *ast.SourceFile, call *ast.CallExpression) *ast.Symbol {
	callee := unwrap(call.Expression)

	var nameNode *ast.Node
	switch {
	case ast.IsIdentifier(callee):
		nameNode = callee
	case ast.IsPropertyAccessExpression(callee):
		access := callee.AsPropertyAccessExpression()
		if access.Expression.Kind != ast.KindThisKeyword {
			return nil
		}
		nameNode = access.Name()
	default:
		return nil
	}

	symbol := typeChecker.GetSymbolAtLocation(nameNode)
	if symbol == nil || len(symbol.Declarations) == 0 {
		return nil
	}
	for _, declaration := range symbol.Declarations {
		if ast.GetSourceFileOfNode(declaration) != sourceFile {
			return nil
		}
		// An import specifier lives in this file but the function does not —
		// wrapping an import is the local seam, which is the point.
		switch declaration.Kind {
		case ast.KindImportSpecifier, ast.KindImportClause, ast.KindNamespaceImport,
			ast.KindImportEqualsDeclaration:
			return nil
		}
	}
	return symbol
}

// A class that declares `implements` is satisfying a contract, so a method that
// only forwards is the contract's shape rather than a redundant wrapper.
func inImplementingClass(node *ast.Node) bool {
	for current := node.Parent; current != nil; current = current.Parent {
		if !ast.IsClassLike(current) {
			continue
		}
		clauses := current.ClassLikeData().HeritageClauses
		if clauses == nil {
			return false
		}
		for _, heritage := range clauses.Nodes {
			if heritage.AsHeritageClause().Token == ast.KindImplementsKeyword {
				return true
			}
		}
		return false
	}
	return false
}

// A single `this.track(<call>)` layer, returning the inner call.
func unwrapTracked(call *ast.Node) *ast.Node {
	callee := call.AsCallExpression().Expression
	if !ast.IsPropertyAccessExpression(callee) {
		return nil
	}
	access := callee.AsPropertyAccessExpression()
	if access.Expression.Kind != ast.KindThisKeyword || access.Name().Text() != "track" {
		return nil
	}
	args := call.AsCallExpression().Arguments
	if args == nil || len(args.Nodes) != 1 {
		return nil
	}
	inner := unwrap(args.Nodes[0])
	if !ast.IsCallExpression(inner) {
		return nil
	}
	return inner
}

// The JS rule only ever collected function declarations, variable initialisers
// and class members, so a method defined inside an object literal was never a
// candidate. Visitor-style literals rely on that.
func inObjectLiteral(node *ast.Node) bool {
	parent := node.Parent
	if parent == nil {
		return false
	}
	if ast.IsObjectLiteralExpression(parent) {
		return true
	}
	return ast.IsPropertyAssignment(parent) && parent.Parent != nil &&
		ast.IsObjectLiteralExpression(parent.Parent)
}

type forwarder struct {
	node   *ast.Node
	target string
}

var NoPassthroughFunctionsRule = rule.Rule{
	Name: "no-passthrough-functions",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		// Keyed by the resolved target symbol: two wrappers onto the same target
		// are a deliberate fan-in (putText/putBinary -> write), not a redundant
		// hop. Only a sole forwarder is a passthrough.
		forwarders := map[*ast.Symbol][]forwarder{}
		order := []*ast.Symbol{}

		check := func(node *ast.Node) {
			if inImplementingClass(node) || inObjectLiteral(node) {
				return
			}

			expression := soleExpression(node.Body())
			if expression == nil {
				return
			}

			call := unwrap(expression)
			if !ast.IsCallExpression(call) {
				return
			}

			// `return this.track(this.send(args))` still forwards to `send` —
			// track only registers the promise, it is not the destination.
			if inner := unwrapTracked(call); inner != nil {
				call = inner
			}

			parameters := parameterNames(node.Parameters())
			if len(parameters) == 0 {
				return
			}
			if !forwardsOnlyParameters(call.AsCallExpression(), parameters) {
				return
			}

			symbol := localTargetSymbol(ctx.TypeChecker, ctx.SourceFile, call.AsCallExpression())
			if symbol == nil {
				return
			}

			reported := node.Name()
			if reported == nil {
				reported = node
			}
			if _, seen := forwarders[symbol]; !seen {
				order = append(order, symbol)
			}
			forwarders[symbol] = append(forwarders[symbol], forwarder{node: reported, target: symbol.Name})
		}

		return rule.RuleListeners{
			ast.KindFunctionDeclaration: check,
			ast.KindFunctionExpression:  check,
			ast.KindArrowFunction:       check,
			ast.KindMethodDeclaration:   check,

			rule.ListenerOnExit(ast.KindSourceFile): func(node *ast.Node) {
				for _, symbol := range order {
					found := forwarders[symbol]
					if len(found) != 1 {
						continue
					}
					ctx.ReportNode(found[0].node, buildPassthroughMessage(found[0].target))
				}
			},
		}
	},
}
