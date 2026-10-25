package require_object_params

import (
	"strconv"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func buildTooManyParamsMessage(label string, count int) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "tooManyParams",
		Description: label + " has " + strconv.Itoa(count) + " parameters.",
		Help:        "Use a single object parameter instead, so every call site names what it passes.",
	}
}

func buildMissingSignalMessage(label string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "missingSignal",
		Description: label + " is async but its object parameter has no required `AbortSignal` property.",
		Help:        "Take `{ signal }: { readonly signal: AbortSignal }` so every caller can cancel it.",
	}
}

// A `this` parameter is a type annotation, not an argument the caller passes.
func countRealParams(parameters []*ast.Node) int {
	count := 0
	for _, parameter := range parameters {
		name := parameter.Name()
		if name != nil && ast.IsIdentifier(name) && name.Text() == "this" {
			continue
		}
		count++
	}
	return count
}

func realParams(parameters []*ast.Node) []*ast.Node {
	params := make([]*ast.Node, 0, len(parameters))
	for _, parameter := range parameters {
		name := parameter.Name()
		if name != nil && ast.IsIdentifier(name) && name.Text() == "this" {
			continue
		}
		params = append(params, parameter)
	}
	return params
}

// A body-less declaration is an overload or ambient; an `override` or a
// `[Symbol.asyncDispose]()` has its shape dictated by someone else, and so does
// an object-literal method, which is always an implementation of a contract.
func signatureIsOurs(ctx rule.RuleContext, node *ast.Node) bool {
	if node.Body() == nil || ast.HasAbstractModifier(node) ||
		ast.HasSyntacticModifier(node, ast.ModifierFlagsOverride) {
		return false
	}
	if ast.IsMethodDeclaration(node) {
		if name := node.Name(); name != nil && ast.IsComputedPropertyName(name) {
			return false
		}
		if node.Parent != nil && ast.IsObjectLiteralExpression(node.Parent) {
			return false
		}
		if implementsLibraryMember(ctx, node) {
			return false
		}
	}
	return true
}

// A method that fills in a member a library declared — kysely's
// `Driver.acquireConnection()` through `implements`, or
// `DialectAdapterBase.acquireMigrationLock()` through `extends` — has the shape
// that library chose. A member our own code declared is checked where it is
// declared, so it does not exempt the method.
func implementsLibraryMember(ctx rule.RuleContext, node *ast.Node) bool {
	class := node.Parent
	name := node.Name()
	if class == nil || name == nil || !ast.IsIdentifier(name) {
		return false
	}
	for _, kind := range []ast.Kind{ast.KindImplementsKeyword, ast.KindExtendsKeyword} {
		clause := ast.GetHeritageClause(class, kind)
		if clause == nil {
			continue
		}
		for _, heritage := range clause.AsHeritageClause().Types.Nodes {
			heritageType := ctx.TypeChecker.GetTypeAtLocation(heritage)
			if heritageType == nil {
				continue
			}
			member := checker.Checker_getPropertyOfType(ctx.TypeChecker, heritageType, name.Text())
			if member != nil && declaredOnlyInDeclarationFiles(member) {
				return true
			}
		}
	}
	return false
}

func declaredOnlyInDeclarationFiles(symbol *ast.Symbol) bool {
	if len(symbol.Declarations) == 0 {
		return false
	}
	for _, declaration := range symbol.Declarations {
		file := ast.GetSourceFileOfNode(declaration)
		if file == nil || !file.IsDeclarationFile {
			return false
		}
	}
	return true
}

func isAsync(ctx rule.RuleContext, node *ast.Node) bool {
	if ast.HasSyntacticModifier(node, ast.ModifierFlagsAsync) {
		return true
	}
	signature := ctx.TypeChecker.GetSignatureFromDeclaration(node)
	if signature == nil {
		return false
	}
	returnType := ctx.TypeChecker.GetReturnTypeOfSignature(signature)
	if returnType == nil || utils.IsIntrinsicErrorType(returnType) ||
		utils.IsTypeFlagSet(returnType, checker.TypeFlagsAny|checker.TypeFlagsUnknown) {
		return false
	}
	return utils.IsThenableType(ctx.TypeChecker, node, returnType)
}

func isAbortSignal(t *checker.Type) bool {
	symbol := checker.Type_symbol(t)
	return symbol != nil && symbol.Name == "AbortSignal"
}

// Every part of the parameter's type has to carry a required property typed
// `AbortSignal` — whatever it is called (`signal`, `activityAbortSignal`); one
// part without it is a call site that can still skip cancellation.
func takesRequiredSignal(ctx rule.RuleContext, parameter *ast.Node) bool {
	parameterType := ctx.TypeChecker.GetTypeAtLocation(parameter)
	if parameterType == nil || utils.IsIntrinsicErrorType(parameterType) ||
		utils.IsTypeFlagSet(parameterType, checker.TypeFlagsAny|checker.TypeFlagsUnknown) {
		return true
	}
	for _, part := range utils.UnionTypeParts(parameterType) {
		found := false
		for _, property := range checker.Checker_getPropertiesOfType(ctx.TypeChecker, part) {
			if utils.IsSymbolFlagSet(property, ast.SymbolFlagsOptional) {
				continue
			}
			if isAbortSignal(ctx.TypeChecker.GetTypeOfSymbolAtLocation(property, parameter)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func nameOf(node *ast.Node) string {
	name := node.Name()
	if name == nil {
		return "anonymous"
	}
	if ast.IsIdentifier(name) || ast.IsPrivateIdentifier(name) || ast.IsStringLiteralLike(name) {
		return name.Text()
	}
	return "anonymous"
}

var RequireObjectParamsRule = rule.Rule{
	Name: "require-object-params",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		report := func(node *ast.Node, label string) {
			count := countRealParams(node.Parameters())
			if count > 1 {
				ctx.ReportNode(node, buildTooManyParamsMessage(label, count))
				return
			}
			if ast.IsConstructorDeclaration(node) || !signatureIsOurs(ctx, node) || !isAsync(ctx, node) {
				return
			}
			params := realParams(node.Parameters())
			if len(params) == 0 || !takesRequiredSignal(ctx, params[0]) {
				ctx.ReportNode(node, buildMissingSignalMessage(label))
			}
		}

		// The JS rule only reached an arrow or function expression through a
		// variable declarator, so a callback argument was never checked.
		checkAssigned := func(node *ast.Node) {
			parent := node.Parent
			if parent == nil || !ast.IsVariableDeclaration(parent) {
				return
			}
			report(node, "Function '"+nameOf(parent)+"'")
		}

		return rule.RuleListeners{
			ast.KindArrowFunction: checkAssigned,
			ast.KindConstructor: func(node *ast.Node) {
				report(node, "Constructor")
			},
			ast.KindFunctionDeclaration: func(node *ast.Node) {
				report(node, "Function '"+nameOf(node)+"'")
			},
			ast.KindFunctionExpression: checkAssigned,
			ast.KindMethodDeclaration: func(node *ast.Node) {
				report(node, "Method")
			},
		}
	},
}
