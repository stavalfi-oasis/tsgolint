package require_zod_compile

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/oasis"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

// Methods that consume a schema rather than build one — the chain ends there and
// there is nothing left to compile.
var terminalMethods = map[string]bool{
	"decode":         true,
	"decodeAsync":    true,
	"encode":         true,
	"encodeAsync":    true,
	"parse":          true,
	"parseAsync":     true,
	"safeDecode":     true,
	"safeEncode":     true,
	"safeParse":      true,
	"safeParseAsync": true,
}

// zod helpers that take or describe a schema rather than building one. Their
// results can carry a Zod-ish type, so the type check alone does not exclude
// them.
var nonSchemaRoots = map[string]bool{
	"clone":         true,
	"compile":       true,
	"config":        true,
	"flattenError":  true,
	"formatError":   true,
	"prettifyError": true,
	"registry":      true,
	"toJSONSchema":  true,
	"treeifyError":  true,
	"withParser":    true,
}

func buildRequireCompileMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "requireZodCompile",
		Description: "Every zod schema must be AOT-compiled.",
		Help:        "Wrap the whole schema expression in z.compile(...).",
	}
}

func calleeMethod(node *ast.Node) string {
	if !ast.IsCallExpression(node) {
		return ""
	}
	if callee := node.AsCallExpression().Expression; ast.IsPropertyAccessExpression(callee) {
		return callee.AsPropertyAccessExpression().Name().Text()
	}
	return ""
}

// Walks a call/property chain back to the identifier it started from, which for
// a zod schema is the local binding of the zod namespace (`z`, or whatever the
// import called it).
func rootIdentifier(node *ast.Node) *ast.Node {
	current := node
	for {
		switch {
		case ast.IsCallExpression(current):
			current = current.AsCallExpression().Expression
		case ast.IsPropertyAccessExpression(current):
			current = current.AsPropertyAccessExpression().Expression
		case ast.IsParenthesizedExpression(current):
			current = current.AsParenthesizedExpression().Expression
		case ast.IsIdentifier(current):
			return current
		default:
			return nil
		}
	}
}

// The fix inserts `<root>.compile(`, which only holds when `<root>` really has
// a compile method. A schema reached through a helper (`Storage.#schema()`)
// roots at that helper's owner, and wrapping it in `Storage.compile(` produces
// code that does not compile — so that case reports without a fix.
func compilerName(typeChecker *checker.Checker, node *ast.Node) string {
	root := rootIdentifier(node)
	if root == nil {
		return ""
	}
	t := typeChecker.GetTypeAtLocation(root)
	if t == nil || checker.Checker_getPropertyOfType(typeChecker, t, "compile") == nil {
		return ""
	}
	return root.Text()
}

var RequireZodCompileRule = rule.Rule{
	Name: "require-zod-compile",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				method := calleeMethod(node)
				if nonSchemaRoots[method] || terminalMethods[method] {
					return
				}

				// The call must produce a schema. This replaces the JS rule's
				// bookkeeping of which local names came from a zod import, so a
				// schema reached through a helper or an alias is still seen.
				if !oasis.IsZodSchemaType(ctx.TypeChecker, node) {
					return
				}

				// Only the outermost schema expression is reported — everything
				// inside it is covered by the same compile() call.
				// A schema nested in an object or array literal is a member of
				// the enclosing schema, which carries the compile() for all of
				// them — so the walk has to pass through those containers, not
				// stop at them.
				for parent := node.Parent; parent != nil; parent = parent.Parent {
					if calleeMethod(parent) == "compile" {
						return
					}
					switch {
					case ast.IsPropertyAssignment(parent),
						ast.IsShorthandPropertyAssignment(parent),
						ast.IsObjectLiteralExpression(parent),
						ast.IsArrayLiteralExpression(parent),
						ast.IsSpreadElement(parent):
						continue
					case ast.IsPropertyAccessExpression(parent),
						ast.IsParenthesizedExpression(parent),
						ast.IsCallExpression(parent),
						parent.Kind == ast.KindNonNullExpression:
						if oasis.IsZodSchemaType(ctx.TypeChecker, parent) {
							return
						}
					default:
						// Reached a statement or declaration: this is the root.
					}
					if !ast.IsPropertyAssignment(parent) &&
						!ast.IsShorthandPropertyAssignment(parent) &&
						!ast.IsObjectLiteralExpression(parent) &&
						!ast.IsArrayLiteralExpression(parent) &&
						!ast.IsSpreadElement(parent) &&
						!ast.IsPropertyAccessExpression(parent) &&
						!ast.IsParenthesizedExpression(parent) &&
						!ast.IsCallExpression(parent) &&
						parent.Kind != ast.KindNonNullExpression {
						break
					}
				}

				zodName := compilerName(ctx.TypeChecker, node)
				if zodName == "" {
					ctx.ReportNode(node, buildRequireCompileMessage())
					return
				}

				ctx.ReportNodeWithFixes(node, buildRequireCompileMessage(), func() []rule.RuleFix {
					return []rule.RuleFix{
						rule.RuleFixInsertBefore(ctx.SourceFile, node, zodName+".compile("),
						rule.RuleFixInsertAfter(node, ")"),
					}
				})
			},
		}
	},
}
