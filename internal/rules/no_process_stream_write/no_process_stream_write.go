package no_process_stream_write

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

var replacements = map[string]string{
	"stdout": "console.log",
	"stderr": "console.error",
}

func buildStreamWriteMessage(stream string, replacement string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "streamWrite",
		Description: "Use " + replacement + " instead of process." + stream + ".write.",
		Help:        "console methods format values; a raw stream write does not.",
	}
}

// `process` is only node's process when its type says so. The JS rule matched
// the identifier by name, so any local named `process` with a `stdout.write`
// shape was reported.
func isNodeProcess(typeChecker *checker.Checker, node *ast.Node) bool {
	t := utils.GetConstrainedTypeAtLocation(typeChecker, node)
	symbol := checker.Type_symbol(t)
	return symbol != nil && symbol.Name == "Process"
}

var NoProcessStreamWriteRule = rule.Rule{
	Name: "no-process-stream-write",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				call := node.AsCallExpression()
				if !ast.IsPropertyAccessExpression(call.Expression) {
					return
				}

				write := call.Expression.AsPropertyAccessExpression()
				if write.Name().Text() != "write" || !ast.IsPropertyAccessExpression(write.Expression) {
					return
				}

				stream := write.Expression.AsPropertyAccessExpression()
				streamName := stream.Name().Text()
				replacement, banned := replacements[streamName]
				if !banned || !isNodeProcess(ctx.TypeChecker, stream.Expression) {
					return
				}

				ctx.ReportNodeWithFixes(
					node,
					buildStreamWriteMessage(streamName, replacement),
					func() []rule.RuleFix {
						return []rule.RuleFix{
							rule.RuleFixReplace(ctx.SourceFile, call.Expression, replacement),
						}
					},
				)
			},
		}
	},
}
