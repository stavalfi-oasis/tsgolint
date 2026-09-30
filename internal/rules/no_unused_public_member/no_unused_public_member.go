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
				symbol := typeChecker.GetSymbolAtLocation(node)
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
