package no_curl

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

type BannedCommandGroup struct {
	Commands []string `json:"commands"`
	Message  string   `json:"message"`
}

type NoCurlOptions struct {
	ExecFunctions []string             `json:"execFunctions"`
	Groups        []BannedCommandGroup `json:"groups"`
}

var defaultExecFunctions = []string{"execFile", "execFileAsync", "spawn"}

func buildBannedCommandMessage(message string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "bannedCommand",
		Description: message,
	}
}

// `execFile(...)` and `child.execFile(...)` both name the same function here.
func calleeName(node *ast.Node) string {
	callee := node.AsCallExpression().Expression
	if ast.IsIdentifier(callee) {
		return callee.Text()
	}
	if ast.IsPropertyAccessExpression(callee) {
		return callee.AsPropertyAccessExpression().Name().Text()
	}
	return ""
}

var NoCurlRule = rule.Rule{
	Name: "no-curl",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		opts := utils.UnmarshalOptions[NoCurlOptions](options, "no-curl")
		execFunctions := opts.ExecFunctions
		if len(execFunctions) == 0 {
			execFunctions = defaultExecFunctions
		}

		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				name := calleeName(node)
				if name == "" || !slices.Contains(execFunctions, name) {
					return
				}
				args := node.AsCallExpression().Arguments
				if args == nil || len(args.Nodes) == 0 {
					return
				}
				command := args.Nodes[0]
				if !ast.IsStringLiteral(command) {
					return
				}
				for _, group := range opts.Groups {
					if slices.Contains(group.Commands, command.Text()) {
						ctx.ReportNode(command, buildBannedCommandMessage(group.Message))
						return
					}
				}
			},
		}
	},
}
