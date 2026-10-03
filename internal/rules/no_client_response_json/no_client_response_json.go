package no_client_response_json

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

const clientResponseTypeName = "ClientResponse"

func buildClientResponseJsonMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "clientResponseJson",
		Description: "'json()' on a typed hono client response hands back the shape this build was compiled against, unchecked.",
		Help:        "Call 'this.parsedJson({ response, schema })' from 'TypedClient' instead, so the body is validated against the schema this service expects and a service deployed at a different version fails loudly.",
	}
}

// The receiver's type, following unions so `A | ClientResponse<...>` is still
// caught, and aliases so a named alias of the interface is too.
func isClientResponse(typeChecker *checker.Checker, node *ast.Node) bool {
	t := typeChecker.GetTypeAtLocation(node)
	if t == nil {
		return false
	}

	seen := map[*checker.Type]bool{}
	var walk func(t *checker.Type) bool
	walk = func(t *checker.Type) bool {
		if t == nil || seen[t] {
			return false
		}
		seen[t] = true
		if symbol := checker.Type_symbol(t); symbol != nil && symbol.Name == clientResponseTypeName {
			return true
		}
		for _, part := range utils.UnionTypeParts(t) {
			if walk(part) {
				return true
			}
		}
		return false
	}
	return walk(t)
}

var NoClientResponseJsonRule = rule.Rule{
	Name: "no-client-response-json",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				callee := node.AsCallExpression().Expression
				if !ast.IsPropertyAccessExpression(callee) {
					return
				}
				access := callee.AsPropertyAccessExpression()
				if access.Name().Text() != "json" {
					return
				}
				if !isClientResponse(ctx.TypeChecker, access.Expression) {
					return
				}
				ctx.ReportNode(node, buildClientResponseJsonMessage())
			},
		}
	},
}
