package require_async_queue

import (
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/typescript-eslint/tsgolint/internal/oasis"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

const optionName = "asyncQueue"

const optionType = "AsyncQueue"

func buildMissingOptionMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "missingOption",
		Description: "A class extending " + oasis.DisposableBaseName + " must accept '" + optionName + ": " + optionType + "' in its constructor options and pass it to super(): an app owns one queue and hands it to everything it builds.",
		Help:        "Add 'readonly " + optionName + ": " + optionType + ";' to the constructor's options object and call 'super({ " + optionName + " })'.",
	}
}

// The leading whitespace on the line an offset sits on, so inserted code lines
// up with its neighbours.
func indentOf(sourceFile *ast.SourceFile, offset int) string {
	text := sourceFile.Text()
	start := offset
	for start > 0 && text[start-1] != '\n' {
		start--
	}
	end := start
	for end < offset && (text[end] == ' ' || text[end] == '\t') {
		end++
	}
	return text[start:end]
}

func constructorOf(class *ast.Node) *ast.Node {
	for _, member := range class.Members() {
		if ast.IsConstructorDeclaration(member) && member.Body() != nil {
			return member
		}
	}
	return nil
}

// The first parameter a caller actually passes — `this` is a type annotation.
func firstRealParam(node *ast.Node) *ast.Node {
	for _, parameter := range node.Parameters() {
		name := parameter.Name()
		if name != nil && ast.IsIdentifier(name) && name.Text() == "this" {
			continue
		}
		return parameter
	}
	return nil
}

// Only an inline type literal can be fixed: a named interface is shared with
// other declarations and editing it would reach further than this class.
func typeLiteralOf(parameter *ast.Node) *ast.Node {
	typeNode := parameter.Type()
	if typeNode == nil || !ast.IsTypeLiteralNode(typeNode) {
		return nil
	}
	return typeNode
}

func hasMember(typeLiteral *ast.Node, name string) bool {
	for _, member := range typeLiteral.AsTypeLiteralNode().Members.Nodes {
		memberName := member.Name()
		if memberName == nil {
			continue
		}
		if (ast.IsIdentifier(memberName) || ast.IsStringLiteralLike(memberName)) &&
			memberName.Text() == name {
			return true
		}
	}
	return false
}

func bindingHasElement(pattern *ast.Node, name string) bool {
	for _, element := range pattern.AsBindingPattern().Elements.Nodes {
		binding := element.AsBindingElement()
		property := binding.PropertyName
		if property == nil {
			property = binding.Name()
		}
		if property != nil && ast.IsIdentifier(property) && property.Text() == name {
			return true
		}
	}
	return false
}

// The `super(...)` call of a constructor body, which the fix has to widen so
// the queue reaches ADisposable itself.
func superCallOf(body *ast.Node) *ast.Node {
	if body == nil || !ast.IsBlock(body) {
		return nil
	}
	for _, statement := range body.AsBlock().Statements.Nodes {
		if !ast.IsExpressionStatement(statement) {
			continue
		}
		expression := ast.SkipParentheses(statement.AsExpressionStatement().Expression)
		if ast.IsCallExpression(expression) &&
			expression.AsCallExpression().Expression.Kind == ast.KindSuperKeyword {
			return expression
		}
	}
	return nil
}

// The module specifier this file already uses for ADisposable, rewritten to
// point at async-queue.ts. Every class the rule fires on imports ADisposable
// from somewhere and the two modules are siblings, so this resolves correctly
// whether the file spells it "#libs/a-disposable.ts", "./a-disposable.ts" or a
// long relative path — without the rule guessing a path of its own.
func asyncQueueSpecifier(sourceFile *ast.SourceFile) (string, bool) {
	for _, statement := range sourceFile.Statements.Nodes {
		if !ast.IsImportDeclaration(statement) {
			continue
		}
		specifier := statement.AsImportDeclaration().ModuleSpecifier
		if specifier == nil || !ast.IsStringLiteralLike(specifier) {
			continue
		}
		if text := specifier.Text(); strings.Contains(text, "a-disposable") {
			return strings.Replace(text, "a-disposable", "async-queue", 1), true
		}
	}
	return "", false
}

// Reports whether AsyncQueue is already in scope, so the fix does not add a
// second import of it.
func importsAsyncQueue(sourceFile *ast.SourceFile) bool {
	for _, statement := range sourceFile.Statements.Nodes {
		if !ast.IsImportDeclaration(statement) {
			continue
		}
		clause := statement.AsImportDeclaration().ImportClause
		if clause == nil {
			continue
		}
		bindings := clause.AsImportClause().NamedBindings
		if bindings == nil || !ast.IsNamedImports(bindings) {
			continue
		}
		for _, element := range bindings.AsNamedImports().Elements.Nodes {
			if name := element.Name(); name != nil && ast.IsIdentifier(name) && name.Text() == optionType {
				return true
			}
		}
	}
	return false
}

// The import fix, or nothing when AsyncQueue is already imported or the file's
// ADisposable import cannot be found to model the specifier on.
func importFixes(sourceFile *ast.SourceFile) []rule.RuleFix {
	if importsAsyncQueue(sourceFile) {
		return nil
	}
	specifier, ok := asyncQueueSpecifier(sourceFile)
	if !ok {
		return nil
	}
	for _, statement := range sourceFile.Statements.Nodes {
		if !ast.IsImportDeclaration(statement) {
			continue
		}
		moduleSpecifier := statement.AsImportDeclaration().ModuleSpecifier
		if moduleSpecifier == nil || !ast.IsStringLiteralLike(moduleSpecifier) ||
			!strings.Contains(moduleSpecifier.Text(), "a-disposable") {
			continue
		}
		return []rule.RuleFix{
			rule.RuleFixInsertAfter(statement, "\nimport type { "+optionType+" } from \""+specifier+"\";"),
		}
	}
	return nil
}

// Widens `super()` to `super({ asyncQueue })`. A super() that already passes
// something is left alone: the fix cannot know what the extra argument means.
func superFixes(superCall *ast.Node) []rule.RuleFix {
	if superCall == nil {
		return nil
	}
	arguments := superCall.AsCallExpression().Arguments
	if arguments != nil && len(arguments.Nodes) > 0 {
		return nil
	}
	return []rule.RuleFix{
		rule.RuleFixReplaceRange(
			core.NewTextRange(superCall.End()-2, superCall.End()),
			"({ "+optionName+" })",
		),
	}
}

var RequireAsyncQueueRule = rule.Rule{
	Name: "require-async-queue",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		check := func(node *ast.Node) {
			if oasis.IsDisposableBase(node) || !oasis.ExtendsDisposable(ctx.TypeChecker, node) {
				return
			}

			reportOn := node.Name()
			if reportOn == nil {
				reportOn = node
			}

			constructorNode := constructorOf(node)
			if constructorNode == nil {
				ctx.ReportNodeWithFixes(reportOn, buildMissingOptionMessage(), func() []rule.RuleFix {
					return append(insertConstructorFix(ctx.SourceFile, node), importFixes(ctx.SourceFile)...)
				})
				return
			}

			parameter := firstRealParam(constructorNode)
			if parameter == nil {
				ctx.ReportNodeWithFixes(constructorNode, buildMissingOptionMessage(), func() []rule.RuleFix {
					fixes := insertParameterFix(ctx.SourceFile, constructorNode)
					fixes = append(fixes, superFixes(superCallOf(constructorNode.Body()))...)
					return append(fixes, importFixes(ctx.SourceFile)...)
				})
				return
			}

			typeLiteral := typeLiteralOf(parameter)
			if typeLiteral == nil {
				// A named options interface: report, but leave the edit to a
				// human — the interface is shared and may be used elsewhere.
				ctx.ReportNode(parameter, buildMissingOptionMessage())
				return
			}
			if hasMember(typeLiteral, optionName) {
				return
			}

			ctx.ReportNodeWithFixes(parameter, buildMissingOptionMessage(), func() []rule.RuleFix {
				fixes := []rule.RuleFix{insertTypeMemberFix(ctx.SourceFile, typeLiteral)}
				name := parameter.Name()
				if name != nil && ast.IsObjectBindingPattern(name) && !bindingHasElement(name, optionName) {
					fixes = append(fixes, insertBindingElementFix(ctx.SourceFile, name))
				}
				fixes = append(fixes, superFixes(superCallOf(constructorNode.Body()))...)
				return append(fixes, importFixes(ctx.SourceFile)...)
			})
		}

		return rule.RuleListeners{
			ast.KindClassDeclaration: check,
			ast.KindClassExpression:  check,
		}
	},
}

// Adds `readonly asyncQueue: AsyncQueue;` as the first member of the options
// type literal. First, not sorted into place: oxfmt and sort-keys own ordering,
// and `asyncQueue` sorts ahead of nearly everything anyway.
func insertTypeMemberFix(sourceFile *ast.SourceFile, typeLiteral *ast.Node) rule.RuleFix {
	members := typeLiteral.AsTypeLiteralNode().Members.Nodes
	text := "readonly " + optionName + ": " + optionType + ";"
	if len(members) == 0 {
		return rule.RuleFixReplaceRange(
			core.NewTextRange(typeLiteral.End()-1, typeLiteral.End()-1),
			" "+text+" ",
		)
	}
	first := members[0]
	start := scanner.SkipTrivia(sourceFile.Text(), first.Pos())
	return rule.RuleFixInsertBefore(sourceFile, first, text+"\n"+indentOf(sourceFile, start))
}

// Adds `asyncQueue,` as the first element of the destructuring pattern, to
// match the property just added to its type.
func insertBindingElementFix(sourceFile *ast.SourceFile, pattern *ast.Node) rule.RuleFix {
	elements := pattern.AsBindingPattern().Elements.Nodes
	if len(elements) == 0 {
		return rule.RuleFixReplaceRange(
			core.NewTextRange(pattern.End()-1, pattern.End()-1),
			" "+optionName+" ",
		)
	}
	first := elements[0]
	start := scanner.SkipTrivia(sourceFile.Text(), first.Pos())
	return rule.RuleFixInsertBefore(sourceFile, first, optionName+",\n"+indentOf(sourceFile, start))
}

// Gives a parameterless constructor the options object it was missing.
func insertParameterFix(sourceFile *ast.SourceFile, constructorNode *ast.Node) []rule.RuleFix {
	body := constructorNode.Body()
	open := strings.Index(sourceFile.Text()[constructorNode.Pos():body.Pos()], "(")
	if open == -1 {
		return nil
	}
	at := constructorNode.Pos() + open + 1
	return []rule.RuleFix{
		rule.RuleFixReplaceRange(
			core.NewTextRange(at, at),
			"{ "+optionName+" }: { readonly "+optionName+": "+optionType+" }",
		),
	}
}

// Gives a class with no constructor at all one that accepts the queue and hands
// it straight to ADisposable, which is where it is stored.
func insertConstructorFix(sourceFile *ast.SourceFile, class *ast.Node) []rule.RuleFix {
	members := class.Members()
	indent := indentOf(sourceFile, scanner.SkipTrivia(sourceFile.Text(), class.Pos()))
	inner := indent + "  "
	text := "public constructor({\n" +
		inner + "  " + optionName + ",\n" +
		inner + "}: {\n" +
		inner + "  readonly " + optionName + ": " + optionType + ";\n" +
		inner + "}) {\n" +
		inner + "  super({ " + optionName + " });\n" +
		inner + "}"
	if len(members) == 0 {
		return []rule.RuleFix{
			rule.RuleFixReplaceRange(
				core.NewTextRange(class.End()-1, class.End()-1),
				"\n"+inner+text+"\n"+indent,
			),
		}
	}
	first := members[0]
	start := scanner.SkipTrivia(sourceFile.Text(), first.Pos())
	return []rule.RuleFix{
		rule.RuleFixInsertBefore(sourceFile, first, text+"\n\n"+indentOf(sourceFile, start)),
	}
}
