package no_async_global_functions

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

type NoAsyncGlobalFunctionsOptions struct {
	// Function names that may be called directly. Matched against both the
	// called name (`readFile`) and the whole callee text (`fs.readFile`).
	ExcludeFunctions []string `json:"excludeFunctions"`
	// Module specifiers whose every export may be called directly, e.g.
	// "node:fs/promises". Matched exactly against the import's specifier.
	ExcludeModules []string `json:"excludeModules"`
}

func buildAsyncGlobalFunctionMessage(name string, reason string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "asyncGlobalFunction",
		Description: "`" + name + "` " + reason + ", so calling it directly leaves work nothing owns.",
		Help:        "Wrap it in a class that owns the pending work, so disposing the class disposes the work.",
	}
}

// The identifier the callee chain hangs off: `fetch` in `fetch(url)`, `fs` in
// `fs.promises.readFile(p)`. Anything else — `this.x()`, `(await f()).g()`,
// `arr[0].then()` — has no name to resolve, and is not a global call.
func rootIdentifier(expression *ast.Node) *ast.Node {
	for {
		expression = ast.SkipParentheses(expression)
		switch {
		case ast.IsIdentifier(expression):
			return expression
		case ast.IsPropertyAccessExpression(expression):
			expression = expression.AsPropertyAccessExpression().Expression
		default:
			return nil
		}
	}
}

// The text of the callee as written, used for `fs.readFile`-style excludes.
func calleeText(sourceFile *ast.SourceFile, callee *ast.Node) string {
	textRange := utils.TrimNodeTextRange(sourceFile, callee)
	return sourceFile.Text()[textRange.Pos():textRange.End()]
}

// The name actually being called: the last identifier of the chain.
func calleeName(callee *ast.Node) string {
	callee = ast.SkipParentheses(callee)
	if ast.IsIdentifier(callee) {
		return callee.Text()
	}
	if ast.IsPropertyAccessExpression(callee) {
		return callee.AsPropertyAccessExpression().Name().Text()
	}
	return ""
}

// The specifier the binding was imported from, or "" for an ambient global.
// Walking up from the declaration covers every import shape — named, default,
// namespace — without a case per form.
func importModuleSpecifier(declaration *ast.Node) string {
	for node := declaration; node != nil; node = node.Parent {
		switch node.Kind {
		case ast.KindImportDeclaration:
			specifier := node.AsImportDeclaration().ModuleSpecifier
			if specifier != nil && ast.IsStringLiteral(specifier) {
				return specifier.Text()
			}
			return ""
		case ast.KindSourceFile:
			return ""
		}
	}
	return ""
}

// Whether the binding ultimately comes from outside this codebase: an ambient
// global (`fetch`, `setTimeout`, `process`) or a package/builtin import. Both
// land in a `.d.ts`; a binding declared in a `.ts` is the repo's own code and
// is already wrapped by whatever class owns it.
//
// Resolving the alias is what separates the two: `readFile` imported from
// `node:fs/promises` is declared, locally, by an ImportSpecifier in a `.ts`
// file. Only the alias target says where it really lives.
func resolveExternalBinding(typeChecker *checker.Checker, identifier *ast.Node) (bool, string) {
	symbol := typeChecker.GetSymbolAtLocation(identifier)
	if symbol == nil {
		return false, ""
	}

	moduleSpecifier := ""
	if len(symbol.Declarations) > 0 {
		moduleSpecifier = importModuleSpecifier(symbol.Declarations[0])
	}

	// Alias chains can be several hops (a re-export of a re-export), and a
	// cycle would otherwise spin here.
	for range 10 {
		if !utils.IsSymbolFlagSet(symbol, ast.SymbolFlagsAlias) {
			break
		}
		next := checker.Checker_getImmediateAliasedSymbol(typeChecker, symbol)
		if next == nil || next == symbol {
			break
		}
		symbol = next
	}

	if len(symbol.Declarations) == 0 {
		return false, moduleSpecifier
	}
	for _, declaration := range symbol.Declarations {
		sourceFile := ast.GetSourceFileOfNode(declaration)
		if sourceFile == nil || !sourceFile.IsDeclarationFile {
			return false, moduleSpecifier
		}
	}
	return true, moduleSpecifier
}

// Why this call leaks work, or "" when it does not. A promise nothing holds
// outlives the scope that started it; a callback handed to a global outlives it
// too, and neither has an owner to cancel it.
//
// The callback test looks at the arguments actually passed rather than the
// declared parameters: `JSON.stringify(value)` and `JSON.stringify(value, fn)`
// share a signature that accepts a replacer, and only the second one schedules
// anything.
func leakReason(typeChecker *checker.Checker, call *ast.Node) string {
	callExpression := call.AsCallExpression()

	if signature := checker.Checker_getResolvedSignature(typeChecker, call, nil, checker.CheckModeNormal); signature != nil {
		returnType := checker.Checker_getReturnTypeOfSignature(typeChecker, signature)
		if returnType != nil && !utils.IsIntrinsicErrorType(returnType) &&
			utils.IsThenableType(typeChecker, call, returnType) {
			return "returns a promise"
		}
	}

	if callExpression.Arguments == nil {
		return ""
	}
	for _, argument := range callExpression.Arguments.Nodes {
		argumentType := typeChecker.GetTypeAtLocation(argument)
		if argumentType == nil || utils.IsIntrinsicErrorType(argumentType) {
			continue
		}
		for _, part := range utils.UnionTypeParts(argumentType) {
			if len(utils.GetCallSignatures(typeChecker, part)) > 0 {
				return "is handed a callback"
			}
		}
	}
	return ""
}

var NoAsyncGlobalFunctionsRule = rule.Rule{
	Name: "no-async-global-functions",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		opts := utils.UnmarshalOptions[NoAsyncGlobalFunctionsOptions](options, "no-async-global-functions")

		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				callee := node.AsCallExpression().Expression
				identifier := rootIdentifier(callee)
				if identifier == nil {
					return
				}

				name := calleeName(callee)
				if name == "" {
					return
				}
				if slices.Contains(opts.ExcludeFunctions, name) ||
					slices.Contains(opts.ExcludeFunctions, calleeText(ctx.SourceFile, callee)) {
					return
				}

				isExternal, moduleSpecifier := resolveExternalBinding(ctx.TypeChecker, identifier)
				if !isExternal {
					return
				}
				if moduleSpecifier != "" && slices.Contains(opts.ExcludeModules, moduleSpecifier) {
					return
				}

				if reason := leakReason(ctx.TypeChecker, node); reason != "" {
					ctx.ReportNode(callee, buildAsyncGlobalFunctionMessage(name, reason))
				}
			},
		}
	},
}
