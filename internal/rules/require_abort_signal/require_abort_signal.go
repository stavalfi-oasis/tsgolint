package require_abort_signal

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// One watched call shape, as `.oxlintrc.json` spells it. A name can carry more
// than one entry: kysely spells both `execute(options)` (every query builder)
// and `execute(callback, options)` (a transaction).
type CancellableCall struct {
	Name         string `json:"name"`
	OptionName   string `json:"optionName"`
	OptionsIndex int    `json:"optionsIndex"`
	// A name shared with unrelated APIs — kysely's `execute`, which every query
	// builder, schema builder and transaction also spells — only counts when the
	// resolved signature really declares the option. A shorter or parameterless
	// signature there is a different function, not an untyped one.
	RequireDeclaredOption bool `json:"requireDeclaredOption"`
}

type RequireAbortSignalOptions struct {
	Calls []CancellableCall `json:"calls"`
}

// The JS rule keyed purely on the method name, which both over- and under-fired:
// any `.send(...)`/`.on(...)`/`.wait(...)` on any object was reported, and it
// carried a `requiresBareCallee` flag to paper over the worst of it. Here the
// name only narrows the candidates; the signature decides.
// The config is the whole list: a call this codebase has to be able to cancel
// is a fact about the codebase, not about the rule.
func cancellableByName(calls []CancellableCall) map[string][]CancellableCall {
	byName := make(map[string][]CancellableCall, len(calls))
	for _, call := range calls {
		byName[call.Name] = append(byName[call.Name], call)
	}
	return byName
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
func acceptsSignalOption(typeChecker *checker.Checker, call *ast.Node, match CancellableCall) bool {
	index := match.OptionsIndex
	signature := checker.Checker_getResolvedSignature(typeChecker, call, nil, checker.CheckModeNormal)
	if signature == nil {
		return !match.RequireDeclaredOption
	}
	parameters := checker.Signature_parameters(signature)
	if index >= len(parameters) {
		// A resolved signature that is simply shorter than the options index is
		// real evidence this is not the API we mean.
		return !match.RequireDeclaredOption && len(parameters) == 0
	}

	parameterType := typeChecker.GetTypeOfSymbolAtLocation(parameters[index], call)
	if parameterType == nil || utils.IsIntrinsicErrorType(parameterType) ||
		utils.IsTypeFlagSet(parameterType, checker.TypeFlagsAny|checker.TypeFlagsUnknown) {
		return !match.RequireDeclaredOption
	}
	return utils.SomeUnionTypePart(parameterType, func(part *checker.Type) bool {
		return checker.Checker_getPropertyOfType(typeChecker, part, match.OptionName) != nil
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
		opts := utils.UnmarshalOptions[RequireAbortSignalOptions](options, "require-abort-signal")
		cancellable := cancellableByName(opts.Calls)
		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				call := node.AsCallExpression()
				candidates, watched := cancellable[calleeName(call.Expression)]
				if !watched {
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

				missing, found := CancellableCall{}, false
				for _, match := range candidates {
					if !acceptsSignalOption(ctx.TypeChecker, node, match) {
						continue
					}
					if match.OptionsIndex < len(args) {
						given := args[match.OptionsIndex]
						if declaresOption(given, match.OptionName) || isDetached(given) {
							return
						}
					}
					if !found {
						missing, found = match, true
					}
				}
				if !found {
					return
				}

				ctx.ReportNode(node, buildAbortSignalMessage(missing.OptionName))
			},
		}
	},
}
