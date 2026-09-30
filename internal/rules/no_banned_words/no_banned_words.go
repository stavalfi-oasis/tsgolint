package no_banned_words

import (
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

type BannedGroup struct {
	Message string   `json:"message"`
	Words   []string `json:"words"`
}

type NoBannedWordsOptions struct {
	Groups []BannedGroup `json:"groups"`
}

func buildBannedWordMessage(message string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "bannedWord",
		Description: message,
	}
}

// Whole words only, matched case-insensitively — the same split the JS rule did.
func containsWord(value string, word string) bool {
	for _, token := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !(r == '_' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		if token == word {
			return true
		}
	}
	return false
}

var NoBannedWordsRule = rule.Rule{
	Name: "no-banned-words",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		opts := utils.UnmarshalOptions[NoBannedWordsOptions](options, "no-banned-words")

		groups := make([]BannedGroup, 0, len(opts.Groups))
		for _, group := range opts.Groups {
			words := make([]string, 0, len(group.Words))
			for _, word := range group.Words {
				words = append(words, strings.ToLower(word))
			}
			groups = append(groups, BannedGroup{Message: group.Message, Words: words})
		}

		check := func(node *ast.Node, value string) {
			for _, group := range groups {
				if slices.ContainsFunc(group.Words, func(word string) bool {
					return containsWord(value, word)
				}) {
					ctx.ReportNode(node, buildBannedWordMessage(group.Message))
					return
				}
			}
		}

		checkLiteral := func(node *ast.Node) {
			check(node, node.Text())
		}

		return rule.RuleListeners{
			ast.KindNoSubstitutionTemplateLiteral: checkLiteral,
			ast.KindStringLiteral:                 checkLiteral,
			ast.KindTemplateExpression: func(node *ast.Node) {
				expression := node.AsTemplateExpression()
				var joined strings.Builder
				joined.WriteString(expression.Head.Text())
				for _, span := range expression.TemplateSpans.Nodes {
					joined.WriteString(" ")
					joined.WriteString(span.AsTemplateSpan().Literal.Text())
				}
				check(node, joined.String())
			},
		}
	},
}
