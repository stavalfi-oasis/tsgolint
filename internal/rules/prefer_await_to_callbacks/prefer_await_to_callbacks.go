package prefer_await_to_callbacks

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// A method whose callback is a framework hook, not an IO callback: `app.onError((error, c) => ...)`
// on a `Hono` receiver is allowed, `onError` on any other receiver is not.
type AllowedMethod struct {
	Class  string `json:"class"`
	Method string `json:"method"`
}

type PreferAwaitToCallbacksOptions struct {
	Allowed []AllowedMethod `json:"allowed"`
}

var eventMethods = []string{"on", "once", "addEventListener", "removeEventListener"}

var arrayMethods = []string{"map", "every", "forEach", "some", "find", "filter"}

var lodashNames = []string{"_", "lodash", "underscore"}

var callbackNames = []string{"callback", "cb"}

var errorNames = []string{"err", "error"}

func buildPreferAwaitToCallbacksMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "preferAwaitToCallbacks",
		Description: "Prefer `async`/`await` to the callback pattern",
		Help:        "Refactor to use an `async` function with `await` instead of passing callbacks for cleaner error handling and control flow.",
	}
}

func identifierName(node *ast.Node) string {
	if node != nil && ast.IsIdentifier(node) {
		return node.Text()
	}
	return ""
}

func isInsideAwaitOrYield(node *ast.Node) bool {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if parent.Kind == ast.KindAwaitExpression || parent.Kind == ast.KindYieldExpression {
			return true
		}
	}
	return false
}

// The receiver's type, following unions so `Hono | undefined` still matches.
func receiverIsClass(typeChecker *checker.Checker, receiver *ast.Node, class string) bool {
	t := typeChecker.GetTypeAtLocation(receiver)
	if t == nil {
		return false
	}
	for _, part := range utils.UnionTypeParts(t) {
		if symbol := checker.Type_symbol(part); symbol != nil && symbol.Name == class {
			return true
		}
	}
	return false
}

func isAllowed(typeChecker *checker.Checker, access *ast.PropertyAccessExpression, allowed []AllowedMethod) bool {
	method := access.Name().Text()
	return slices.ContainsFunc(allowed, func(entry AllowedMethod) bool {
		return entry.Method == method && receiverIsClass(typeChecker, access.Expression, entry.Class)
	})
}

func lastCallbackParameter(node *ast.Node) *ast.Node {
	parameters := node.Parameters()
	if len(parameters) == 0 {
		return nil
	}
	last := parameters[len(parameters)-1]
	if !slices.Contains(callbackNames, identifierName(last.Name())) {
		return nil
	}
	return last
}

var PreferAwaitToCallbacksRule = rule.Rule{
	Name: "prefer-await-to-callbacks",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		opts := utils.UnmarshalOptions[PreferAwaitToCallbacksOptions](options, "prefer-await-to-callbacks")

		checkFunction := func(node *ast.Node) {
			if parameter := lastCallbackParameter(node); parameter != nil {
				ctx.ReportNode(parameter, buildPreferAwaitToCallbacksMessage())
			}
		}

		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				call := node.AsCallExpression()
				calleeName := identifierName(call.Expression)
				if slices.Contains(callbackNames, calleeName) {
					ctx.ReportNode(node, buildPreferAwaitToCallbacksMessage())
					return
				}
				if call.Arguments == nil || len(call.Arguments.Nodes) == 0 {
					return
				}
				arguments := call.Arguments.Nodes
				lastArgument := arguments[len(arguments)-1]
				if !ast.IsFunctionExpression(lastArgument) && !ast.IsArrowFunction(lastArgument) {
					return
				}

				propertyName := ""
				isLodash := false
				if ast.IsPropertyAccessExpression(call.Expression) {
					access := call.Expression.AsPropertyAccessExpression()
					propertyName = access.Name().Text()
					if slices.Contains(eventMethods, propertyName) || isAllowed(ctx.TypeChecker, access, opts.Allowed) {
						return
					}
					isLodash = slices.Contains(lodashNames, identifierName(access.Expression))
				}
				callsArrayMethod := slices.Contains(arrayMethods, propertyName) &&
					(len(arguments) == 1 || (len(arguments) == 2 && isLodash))
				isArrayFunction := slices.Contains(arrayMethods, calleeName) && len(arguments) == 2
				if callsArrayMethod || isArrayFunction {
					return
				}

				parameters := lastArgument.Parameters()
				if len(parameters) == 0 || !slices.Contains(errorNames, identifierName(parameters[0].Name())) {
					return
				}
				if isInsideAwaitOrYield(node) {
					return
				}
				ctx.ReportNode(lastArgument, buildPreferAwaitToCallbacksMessage())
			},
			ast.KindFunctionDeclaration: checkFunction,
			ast.KindFunctionExpression:  checkFunction,
			ast.KindArrowFunction:       checkFunction,
			ast.KindMethodDeclaration:   checkFunction,
			ast.KindConstructor:         checkFunction,
			ast.KindSetAccessor:         checkFunction,
		}
	},
}
