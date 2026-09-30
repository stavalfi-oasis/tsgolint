package async_dispose_last

import (
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/typescript-eslint/tsgolint/internal/oasis"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func buildMustBeLastMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "asyncDisposeNotLast",
		Description: "'[Symbol.asyncDispose]' must be the last member of the class.",
		Help:        "Move it below every other member, so disposal is the final thing the file declares.",
	}
}

// The leading whitespace of the line an offset sits on, so the moved member
// lines up with the ones it is being placed after.
func indentOf(text string, offset int) string {
	start := offset
	for start > 0 && text[start-1] != '\n' {
		start--
	}
	end := start
	for end < offset && (text[end] == ' ' || text[end] == '\t') {
		end++
	}
	return text[start:end]
}

var AsyncDisposeLastRule = rule.Rule{
	Name: "async-dispose-last",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		text := ctx.SourceFile.Text()

		check := func(node *ast.Node) {
			members := node.Members()
			index := -1
			for i, member := range members {
				if oasis.IsAsyncDisposeMember(member) {
					index = i
					break
				}
			}
			if index == -1 || index == len(members)-1 {
				return
			}

			member := members[index]
			next := members[index+1]
			last := members[len(members)-1]

			ctx.ReportNodeWithFixes(member, buildMustBeLastMessage(), func() []rule.RuleFix {
				start := utils.TrimNodeTextRange(ctx.SourceFile, member).Pos()
				cut := utils.TrimNodeTextRange(ctx.SourceFile, next).Pos()
				moved := strings.TrimSpace(text[start:member.End()])
				return []rule.RuleFix{
					rule.RuleFixRemoveRange(core.NewTextRange(start, cut)),
					rule.RuleFixInsertAfter(last, "\n\n"+indentOf(text, start)+moved),
				}
			})
		}

		return rule.RuleListeners{
			ast.KindClassDeclaration: check,
			ast.KindClassExpression:  check,
		}
	},
}
