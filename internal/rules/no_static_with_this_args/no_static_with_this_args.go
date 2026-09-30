package no_static_with_this_args

import (
	"sort"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildStaticWithThisArgsMessage(name string, signature string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "staticWithThisArgs",
		Description: "'" + name + "' is static but every call hands it the same instance state (" + signature + ").",
		Help:        "A static that always needs instance state is just an instance method whose receiver is passed by hand — drop 'static' and read the state from 'this' inside it.",
	}
}

type callSite struct {
	node      *ast.Node
	signature string
}

// True for `this`, and for any member chain rooted at `this` — `this.track`,
// `this.#logger.child`.
func isThisRooted(node *ast.Node) bool {
	current := node
	for ast.IsPropertyAccessExpression(current) {
		current = current.AsPropertyAccessExpression().Expression
	}
	for ast.IsElementAccessExpression(current) {
		current = current.AsElementAccessExpression().Expression
	}
	return current.Kind == ast.KindThisKeyword
}

// Collects the source text of every instance-state reference inside one
// argument. A nested function body is opaque: its `this` is its own.
func thisTexts(sourceFile *ast.SourceFile, root *ast.Node) []string {
	texts := []string{}

	var walk func(node *ast.Node) bool
	walk = func(node *ast.Node) bool {
		if node == nil {
			return false
		}
		if ast.IsFunctionExpression(node) || ast.IsFunctionDeclaration(node) {
			return false
		}
		if node.Kind == ast.KindThisKeyword ||
			((ast.IsPropertyAccessExpression(node) || ast.IsElementAccessExpression(node)) && isThisRooted(node)) {
			texts = append(texts, strings.TrimSpace(sourceFile.Text()[node.Pos():node.End()]))
			return false
		}
		node.ForEachChild(walk)
		return false
	}
	walk(root)

	sort.Strings(texts)
	return texts
}

// The per-call fingerprint of instance state handed over, or "" when the call
// passes none.
func callSignature(sourceFile *ast.SourceFile, call *ast.CallExpression) string {
	if call.Arguments == nil {
		return ""
	}

	parts := make([]string, 0, len(call.Arguments.Nodes))
	any := false
	for _, argument := range call.Arguments.Nodes {
		joined := strings.Join(thisTexts(sourceFile, argument), " + ")
		if joined != "" {
			any = true
		}
		parts = append(parts, joined)
	}
	if !any {
		return ""
	}
	return strings.Join(parts, ", ")
}

// Resolves the callee to the symbol it actually refers to and checks that its
// declaration is a static class member. The JS rule keyed on the literal text
// `ClassName.member`, so an aliased or imported class was invisible to it and
// two same-named classes were conflated.
func staticMemberSymbol(typeChecker *checker.Checker, call *ast.Node) *ast.Symbol {
	callee := call.AsCallExpression().Expression
	if !ast.IsPropertyAccessExpression(callee) {
		return nil
	}

	symbol := typeChecker.GetSymbolAtLocation(callee.AsPropertyAccessExpression().Name())
	if symbol == nil || len(symbol.Declarations) == 0 {
		return nil
	}
	for _, declaration := range symbol.Declarations {
		if !ast.HasSyntacticModifier(declaration, ast.ModifierFlagsStatic) {
			return nil
		}
		if parent := declaration.Parent; parent == nil || !ast.IsClassLike(parent) {
			return nil
		}
	}
	return symbol
}

func displayName(symbol *ast.Symbol) string {
	declaration := symbol.Declarations[0]
	if parent := declaration.Parent; parent != nil {
		if name := parent.Name(); name != nil {
			return name.Text() + "." + symbol.Name
		}
	}
	return symbol.Name
}

var NoStaticWithThisArgsRule = rule.Rule{
	Name: "no-static-with-this-args",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		sites := map[*ast.Symbol][]callSite{}
		order := []*ast.Symbol{}

		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				symbol := staticMemberSymbol(ctx.TypeChecker, node)
				if symbol == nil {
					return
				}
				if _, seen := sites[symbol]; !seen {
					order = append(order, symbol)
				}
				sites[symbol] = append(sites[symbol], callSite{
					node:      node,
					signature: callSignature(ctx.SourceFile, node.AsCallExpression()),
				})
			},
			rule.ListenerOnExit(ast.KindSourceFile): func(node *ast.Node) {
				for _, symbol := range order {
					calls := sites[symbol]
					signature := calls[0].signature
					if signature == "" {
						continue
					}

					// Only report when *every* call hands over the same state —
					// one call that doesn't means the static is genuinely static.
					same := true
					for _, call := range calls {
						if call.signature != signature {
							same = false
							break
						}
					}
					if !same {
						continue
					}

					name := displayName(symbol)
					for _, call := range calls {
						ctx.ReportNode(call.node, buildStaticWithThisArgsMessage(name, signature))
					}
				}
			},
		}
	},
}
