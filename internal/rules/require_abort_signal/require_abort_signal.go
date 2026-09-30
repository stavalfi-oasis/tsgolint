package require_abort_signal

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

type cancellableCall struct {
	optionName   string
	optionsIndex int
}

// The JS rule keyed purely on the method name, which both over- and under-fired:
// any `.send(...)`/`.on(...)`/`.wait(...)` on any object was reported, and it
// carried a `requiresBareCallee` flag to paper over the worst of it. Here the
// name only narrows the candidates; the signature decides.
var cancellable = map[string]cancellableCall{
	"appendFile":      {optionName: "signal", optionsIndex: 2},
	"createInterface": {optionName: "signal", optionsIndex: 0},
	"exec":            {optionName: "signal", optionsIndex: 1},
	"execAsync":       {optionName: "signal", optionsIndex: 1},
	"execFile":        {optionName: "signal", optionsIndex: 2},
	"execFileAsync":   {optionName: "signal", optionsIndex: 2},
	"fetch":           {optionName: "signal", optionsIndex: 1},
	"on":              {optionName: "signal", optionsIndex: 2},
	"once":            {optionName: "signal", optionsIndex: 2},
	"readFile":        {optionName: "signal", optionsIndex: 1},
	"send":            {optionName: "abortSignal", optionsIndex: 1},
	"spawn":           {optionName: "signal", optionsIndex: 2},
	"wait":            {optionName: "signal", optionsIndex: 1},
	"watch":           {optionName: "signal", optionsIndex: 1},
	"writeFile":       {optionName: "signal", optionsIndex: 2},
}

func buildAbortSignalMessage(optionName string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "requireAbortSignal",
		Description: "Pass a signal so this is cancellable.",
		Help:        "Add `" + optionName + "` to the options object.",
	}
}

func calleeName(expression *ast.Node) string {
	if ast.IsIdentifier(expression) {
		return expression.Text()
	}
	if ast.IsPropertyAccessExpression(expression) {
		return expression.AsPropertyAccessExpression().Name().Text()
	}
	return ""
}

// Whether the parameter at optionsIndex is an options bag carrying the signal
// property. That is what separates node's `fetch(url, { signal })` from an
// unrelated local `fetch(a, b)`, which no name match can do.
//
// Types only ever *narrow* here. When the signature cannot be resolved — an
// untyped or `@ts-nocheck` file, an undeclared global — there is nothing to
// narrow with, so the name match stands and coverage is unchanged. Suppressing
// on unresolved types would make the rule quietly weaker on exactly the loose
// code that needs it most.
func acceptsSignalOption(typeChecker *checker.Checker, call *ast.Node, index int, optionName string) bool {
	signature := checker.Checker_getResolvedSignature(typeChecker, call, nil, checker.CheckModeNormal)
	if signature == nil {
		return true
	}
	parameters := checker.Signature_parameters(signature)
	if index >= len(parameters) {
		// A resolved signature that is simply shorter than the options index is
		// real evidence this is not the API we mean.
		return len(parameters) == 0
	}

	parameterType := typeChecker.GetTypeOfSymbolAtLocation(parameters[index], call)
	if parameterType == nil || utils.IsIntrinsicErrorType(parameterType) ||
		utils.IsTypeFlagSet(parameterType, checker.TypeFlagsAny|checker.TypeFlagsUnknown) {
		return true
	}
	return utils.SomeUnionTypePart(parameterType, func(part *checker.Type) bool {
		return checker.Checker_getPropertyOfType(typeChecker, part, optionName) != nil
	})
}

// A literal options object already naming the option is fine; so is anything
// that is not an object literal, since its contents are unknowable here.
func declaresOption(node *ast.Node, optionName string) bool {
	if !ast.IsObjectLiteralExpression(node) {
		return true
	}
	for _, property := range node.AsObjectLiteralExpression().Properties.Nodes {
		if property.Kind == ast.KindSpreadAssignment {
			return true
		}
		if name := property.Name(); name != nil && name.Text() == optionName {
			return true
		}
	}
	return false
}

// `{ detached: true }` opts out deliberately — the child outlives this process.
func isDetached(node *ast.Node) bool {
	if !ast.IsObjectLiteralExpression(node) {
		return false
	}
	for _, property := range node.AsObjectLiteralExpression().Properties.Nodes {
		name := property.Name()
		if name == nil || name.Text() != "detached" {
			continue
		}
		if initializer := property.Initializer(); initializer != nil && initializer.Kind == ast.KindTrueKeyword {
			return true
		}
	}
	return false
}

var RequireAbortSignalRule = rule.Rule{
	Name: "require-abort-signal",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				call := node.AsCallExpression()
				match, watched := cancellable[calleeName(call.Expression)]
				if !watched {
					return
				}

				if !acceptsSignalOption(ctx.TypeChecker, node, match.optionsIndex, match.optionName) {
					return
				}

				var args []*ast.Node
				if call.Arguments != nil {
					args = call.Arguments.Nodes
				}
				for _, argument := range args {
					if argument.Kind == ast.KindSpreadElement {
						return
					}
				}

				if match.optionsIndex < len(args) {
					given := args[match.optionsIndex]
					if declaresOption(given, match.optionName) || isDetached(given) {
						return
					}
				}

				ctx.ReportNode(node, buildAbortSignalMessage(match.optionName))
			},
		}
	},
}
