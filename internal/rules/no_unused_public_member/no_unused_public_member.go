package no_unused_public_member

import (
	"sync"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/compiler"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildUnusedPublicMessage(name string, kind string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "unusedPublicMember",
		Description: "'" + name + "' is " + kind + " but nothing outside the class uses it.",
		Help:        "Make it '#" + name + "' so the class states its real surface, or delete it if nothing calls it at all.",
	}
}

// A member that can be referenced by name from outside, and whose visibility is
// the author's choice rather than a contract's.
func isCandidate(member *ast.Node) bool {
	if !ast.IsMethodDeclaration(member) {
		return false
	}
	if ast.HasSyntacticModifier(member, ast.ModifierFlagsPrivate|ast.ModifierFlagsProtected|ast.ModifierFlagsAbstract) {
		return false
	}
	name := member.Name()
	if name == nil || ast.IsPrivateIdentifier(name) {
		return false
	}
	// A computed name is never referenced by that name — `[Symbol.asyncDispose]`
	// is called by `await using`, and nothing spells it out.
	return ast.IsIdentifier(name) || ast.IsStringLiteralLike(name)
}

// Whether a base class or an implemented interface already declares this name.
// Such a member is the contract's shape, so its visibility is not a free choice.
func satisfiesHeritage(typeChecker *checker.Checker, class *ast.Node, name string) bool {
	clauses := class.ClassLikeData().HeritageClauses
	if clauses == nil {
		return false
	}
	for _, heritage := range clauses.Nodes {
		for _, typeNode := range heritage.AsHeritageClause().Types.Nodes {
			t := typeChecker.GetTypeAtLocation(typeNode)
			if t == nil {
				continue
			}
			if checker.Checker_getPropertyOfType(typeChecker, t, name) != nil {
				return true
			}
		}
	}
	return false
}

// The class a node is written inside, or nil at module scope.
func enclosingClass(node *ast.Node) *ast.Node {
	for current := node.Parent; current != nil; current = current.Parent {
		if ast.IsClassLike(current) {
			return current
		}
	}
	return nil
}

type usage struct {
	// Member *declarations* referenced from outside their own class body,
	// anywhere in the program. Keyed by declaration node, never by symbol: each
	// linter worker runs its own checker and those hand back distinct
	// *ast.Symbol values for the same member, so a symbol-keyed index silently
	// misses every lookup made by a different worker than the one that built it.
	// The AST is shared, so declaration nodes are not.
	external map[*ast.Node]bool
}

// One index per program. Every file's run needs program-wide answers, so
// building it per file would be quadratic; building it once is the same total
// work the lint already does.
var (
	indexMutex sync.Mutex
	indexes    = map[*compiler.Program]*usage{}
)

func programUsage(typeChecker *checker.Checker, program *compiler.Program) *usage {
	indexMutex.Lock()
	defer indexMutex.Unlock()

	if found, ok := indexes[program]; ok {
		return found
	}

	// Pass one is syntactic: collect the names worth resolving. Resolving every
	// identifier in the program would cost far more than the rule is worth.
	names := map[string]bool{}
	for _, sourceFile := range program.SourceFiles() {
		var collect func(node *ast.Node) bool
		collect = func(node *ast.Node) bool {
			if ast.IsClassLike(node) {
				for _, member := range node.Members() {
					if isCandidate(member) {
						names[member.Name().Text()] = true
					}
				}
			}
			node.ForEachChild(collect)
			return false
		}
		collect(sourceFile.AsNode())
	}

	found := &usage{external: map[*ast.Node]bool{}}
	for _, sourceFile := range program.SourceFiles() {
		var walk func(node *ast.Node) bool
		walk = func(node *ast.Node) bool {
			if ast.IsIdentifier(node) && names[node.Text()] {
				symbol := destructuredMember(typeChecker, node)
				if symbol == nil {
					symbol = typeChecker.GetSymbolAtLocation(node)
				}
				if symbol != nil {
					owner := declaringClass(symbol)
					// The declaration itself is not a use, and neither is a
					// reference from inside the very class that declares it.
					if owner != nil && enclosingClass(node) != owner {
						for _, declaration := range symbol.Declarations {
							found.external[declaration] = true
						}
					}
				}
			}
			node.ForEachChild(walk)
			return false
		}
		walk(sourceFile.AsNode())
	}

	indexes[program] = found
	return found
}

// The type a destructuring pattern pulls from, so a member taken off a class by
// `const { run } = Service` can be resolved back to the class.
func bindingSourceType(typeChecker *checker.Checker, element *ast.Node) *checker.Type {
	pattern := element.Parent
	if pattern == nil || pattern.Kind != ast.KindObjectBindingPattern {
		return nil
	}
	owner := pattern.Parent
	if owner == nil {
		return nil
	}
	switch owner.Kind {
	case ast.KindVariableDeclaration:
		if initializer := owner.Initializer(); initializer != nil {
			return typeChecker.GetTypeAtLocation(initializer)
		}
	case ast.KindParameter:
		return typeChecker.GetTypeAtLocation(owner)
	}
	return nil
}

// The member a destructuring binding names, or nil when this identifier is not
// one. In `const { run } = Service` the identifier's own symbol is the new
// variable, so counting it plainly misses that `run` was taken off the class.
func destructuredMember(typeChecker *checker.Checker, node *ast.Node) *ast.Symbol {
	element := node.Parent
	if element == nil || element.Kind != ast.KindBindingElement {
		return nil
	}
	binding := element.AsBindingElement()
	if binding.PropertyName != nil && binding.PropertyName != node {
		return nil
	}
	if binding.PropertyName == nil && binding.Name() != node {
		return nil
	}
	source := bindingSourceType(typeChecker, element)
	if source == nil {
		return nil
	}
	return checker.Checker_getPropertyOfType(typeChecker, source, node.Text())
}

// `class X {}` followed by `export { X }` leaves no export modifier on the
// class, so the modifier check alone would call it local.
func isNamedExport(sourceFile *ast.SourceFile, class *ast.Node) bool {
	name := class.Name()
	if name == nil {
		return false
	}
	for _, statement := range sourceFile.Statements.Nodes {
		if statement.Kind != ast.KindExportDeclaration {
			continue
		}
		clause := statement.AsExportDeclaration().ExportClause
		if clause == nil || clause.Kind != ast.KindNamedExports {
			continue
		}
		for _, element := range clause.AsNamedExports().Elements.Nodes {
			specifier := element.AsExportSpecifier()
			local := specifier.PropertyName
			if local == nil {
				local = specifier.Name()
			}
			if local != nil && local.Text() == name.Text() {
				return true
			}
		}
	}
	return false
}

// The class node a symbol's declarations sit in, or nil when the symbol is not
// a class member at all.
func declaringClass(symbol *ast.Symbol) *ast.Node {
	var owner *ast.Node
	for _, declaration := range symbol.Declarations {
		parent := declaration.Parent
		if parent == nil || !ast.IsClassLike(parent) {
			return nil
		}
		if owner != nil && owner != parent {
			return nil
		}
		owner = parent
	}
	return owner
}

var NoUnusedPublicMemberRule = rule.Rule{
	Name: "no-unused-public-member",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		check := func(node *ast.Node) {
			// An exported class can be used from a file this program does not
			// contain — oxlint builds one program per tsconfig project, so a
			// shared library never sees the apps that consume it. Absence of a
			// use is only evidence when every possible user is in view.
			if ast.HasSyntacticModifier(node, ast.ModifierFlagsExport) || isNamedExport(ctx.SourceFile, node) {
				return
			}
			for _, member := range node.Members() {
				if !isCandidate(member) {
					continue
				}
				name := member.Name()
				if satisfiesHeritage(ctx.TypeChecker, node, name.Text()) {
					continue
				}
				if programUsage(ctx.TypeChecker, ctx.Program).external[member] {
					continue
				}

				kind := "public"
				if ast.HasSyntacticModifier(member, ast.ModifierFlagsStatic) {
					kind = "public and static"
				}
				ctx.ReportNode(name, buildUnusedPublicMessage(name.Text(), kind))
			}
		}

		return rule.RuleListeners{
			ast.KindClassDeclaration: check,
			ast.KindClassExpression:  check,
		}
	},
}
