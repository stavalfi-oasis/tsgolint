package require_async_queue

import (
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/typescript-eslint/tsgolint/internal/oasis"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

const optionName = "asyncQueue"

const optionType = "AsyncQueue"

func buildMissingOptionMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "missingOption",
		Description: "A class extending " + oasis.DisposableBaseName + " must accept '" + optionName + ": " + optionType + "' in its constructor options: an app owns one queue and hands it to everything it builds.",
		Help:        "Add 'readonly " + optionName + ": " + optionType + ";' to the constructor's options object.",
	}
}

const fieldName = "#" + optionName

func buildMissingFieldMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "missingField",
		Description: "The '" + optionName + "' option must be kept on the instance as '" + fieldName + "': a queue that is accepted and dropped is the same as not having one.",
		Help:        "Add 'readonly " + fieldName + ": " + optionType + ";' and assign it in the constructor.",
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

// Reports whether the options parameter's type carries the option, however it
// is written: an inline literal, a named interface, or an intersection of both.
func typeHasOption(typeChecker *checker.Checker, parameter *ast.Node) bool {
	t := utils.GetConstrainedTypeAtLocation(typeChecker, parameter)
	if t == nil {
		return false
	}
	return checker.Checker_getPropertyOfType(typeChecker, t, optionName) != nil
}

// Reports whether the class already declares `readonly #asyncQueue`.
func hasQueueField(class *ast.Node) bool {
	for _, member := range class.Members() {
		if !ast.IsPropertyDeclaration(member) {
			continue
		}
		name := member.Name()
		if name != nil && ast.IsPrivateIdentifier(name) && name.Text() == fieldName {
			return true
		}
	}
	return false
}

// Reports whether the constructor body assigns the option to that field.
func assignsQueueField(body *ast.Node) bool {
	if body == nil || !ast.IsBlock(body) {
		return false
	}
	for _, statement := range body.AsBlock().Statements.Nodes {
		if !ast.IsExpressionStatement(statement) {
			continue
		}
		expression := ast.SkipParentheses(statement.AsExpressionStatement().Expression)
		if !ast.IsBinaryExpression(expression) {
			continue
		}
		binary := expression.AsBinaryExpression()
		if binary.OperatorToken.Kind != ast.KindEqualsToken {
			continue
		}
		left := ast.SkipParentheses(binary.Left)
		if !ast.IsPropertyAccessExpression(left) {
			continue
		}
		access := left.AsPropertyAccessExpression()
		if access.Expression.Kind == ast.KindThisKeyword && access.Name().Text() == fieldName {
			return true
		}
	}
	return false
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
					return append(insertParameterFix(ctx.SourceFile, constructorNode), importFixes(ctx.SourceFile)...)
				})
				return
			}

			// Asking the checker rather than reading the annotation is what
			// covers a named options interface and an intersection — both of
			// which carry the property without spelling it on the parameter.
			if !typeHasOption(ctx.TypeChecker, parameter) {
				typeLiteral := typeLiteralOf(parameter)
				if typeLiteral == nil {
					// A named interface is shared with other declarations, so
					// adding the option to it would reach past this class.
					ctx.ReportNode(parameter, buildMissingOptionMessage())
					return
				}
				ctx.ReportNodeWithFixes(parameter, buildMissingOptionMessage(), func() []rule.RuleFix {
					fixes := []rule.RuleFix{insertTypeMemberFix(ctx.SourceFile, typeLiteral)}
					name := parameter.Name()
					if name != nil && ast.IsObjectBindingPattern(name) && !bindingHasElement(name, optionName) {
						fixes = append(fixes, insertBindingElementFix(ctx.SourceFile, name))
					}
					fixes = append(fixes, storeFixes(ctx.SourceFile, node, constructorNode)...)
					return append(fixes, importFixes(ctx.SourceFile)...)
				})
				return
			}

			// The option is accepted; it still has to be kept.
			if hasQueueField(node) && assignsQueueField(constructorNode.Body()) {
				return
			}
			ctx.ReportNodeWithFixes(constructorNode, buildMissingFieldMessage(), func() []rule.RuleFix {
				return append(storeFixes(ctx.SourceFile, node, constructorNode), importFixes(ctx.SourceFile)...)
			})
		}

		return rule.RuleListeners{
			ast.KindClassDeclaration: check,
			ast.KindClassExpression:  check,
		}
	},
}

// Declares the field, assigns it, and exposes it. The getter is not decoration:
// `no-unused-private-class-members` takes no options, so a class that stores the
// queue and has no use for it yet would otherwise be reported for the field the
// fix just added.
func storeFixes(sourceFile *ast.SourceFile, class *ast.Node, constructorNode *ast.Node) []rule.RuleFix {
	source := optionName
	if parameter := firstRealParam(constructorNode); parameter != nil {
		if name := parameter.Name(); name != nil && ast.IsIdentifier(name) {
			// Not destructured, so the option is only reachable through the
			// parameter it arrived on.
			source = name.Text() + "." + optionName
		}
	}
	fixes := []rule.RuleFix{}
	indent := indentOf(sourceFile, scanner.SkipTrivia(sourceFile.Text(), class.Pos())) + "  "

	if !hasQueueField(class) {
		members := class.Members()
		if len(members) > 0 {
			first := members[0]
			start := scanner.SkipTrivia(sourceFile.Text(), first.Pos())
			fixes = append(fixes, rule.RuleFixInsertBefore(
				sourceFile, first,
				"readonly "+fieldName+": "+optionType+";\n\n"+indentOf(sourceFile, start),
			))
		}
		fixes = append(fixes, rule.RuleFixInsertAfter(constructorNode,
			"\n\n"+indent+"public get "+optionName+"(): "+optionType+" {\n"+
				indent+"  return this."+fieldName+";\n"+
				indent+"}",
		))
	}

	body := constructorNode.Body()
	if !assignsQueueField(body) && body != nil && ast.IsBlock(body) {
		assignment := "this." + fieldName + " = " + source + ";"
		statements := body.AsBlock().Statements.Nodes
		if len(statements) > 0 {
			first := statements[0]
			start := scanner.SkipTrivia(sourceFile.Text(), first.Pos())
			fixes = append(fixes, rule.RuleFixInsertAfter(first, "\n"+indentOf(sourceFile, start)+assignment))
		} else {
			// An empty derived-class body has no super() to assign after, and
			// touching `this` before one is a hard error — so write both.
			fixes = append(fixes, rule.RuleFixReplaceRange(
				core.NewTextRange(body.End()-1, body.End()-1),
				"\n"+indent+"  super();\n"+indent+"  "+assignment+"\n"+indent,
			))
		}
	}
	return fixes
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

// Gives a class with no constructor at all one that accepts the queue, together
// with the field it is kept in and the getter that reads it.
func insertConstructorFix(sourceFile *ast.SourceFile, class *ast.Node) []rule.RuleFix {
	members := class.Members()
	indent := indentOf(sourceFile, scanner.SkipTrivia(sourceFile.Text(), class.Pos()))
	inner := indent + "  "
	text := "readonly " + fieldName + ": " + optionType + ";\n\n" +
		inner + "public constructor({\n" +
		inner + "  " + optionName + ",\n" +
		inner + "}: {\n" +
		inner + "  readonly " + optionName + ": " + optionType + ";\n" +
		inner + "}) {\n" +
		inner + "  super();\n" +
		inner + "  this." + fieldName + " = " + optionName + ";\n" +
		inner + "}\n\n" +
		inner + "public get " + optionName + "(): " + optionType + " {\n" +
		inner + "  return this." + fieldName + ";\n" +
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
