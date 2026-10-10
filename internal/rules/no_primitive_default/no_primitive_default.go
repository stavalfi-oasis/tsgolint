package no_primitive_default

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/jsnum"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// Only a compile-time constant. A value computed at runtime — `string`,
// `number`, a call, a template whose substitutions are not themselves known —
// is the caller's data, not a number this code made up.
const constantFlags = checker.TypeFlagsStringLiteral | checker.TypeFlagsNumberLiteral

func buildPrimitiveDefaultMessage(operator string, text string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "primitiveDefault",
		Description: "`" + operator + " " + text + "` substitutes a made-up string/number for a missing value.",
		Help:        "Let the missing value through and handle it where the meaning of \"missing\" is known, or make the source non-nullable. `{}`, `[]`, `\"\"` and `0` are fine.",
	}
}

// `""` and `0` invent no content — they are the identity of the string or the
// sum being built, the same way `[]` is. Every other constant is a guess.
func isBannedConstant(t *checker.Type) bool {
	if !utils.IsTypeFlagSet(t, constantFlags) {
		return false
	}

	literal := t.AsLiteralType()
	if literal == nil {
		return false
	}

	switch value := literal.Value().(type) {
	case string:
		return value != ""
	case jsnum.Number:
		return value != 0
	default:
		return false
	}
}

var NoPrimitiveDefaultRule = rule.Rule{
	Name: "no-primitive-default",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		// A union only counts when every member is a banned constant —
		// `"unknown" | string` carries a runtime branch, and `any` carries no
		// information at all, so reporting either would be guesswork.
		isBanned := func(t *checker.Type) bool {
			return !utils.SomeUnionTypePart(t, func(part *checker.Type) bool {
				return !isBannedConstant(part)
			})
		}

		check := func(node *ast.Node, right *ast.Node, operator string) {
			if !isBanned(utils.GetConstrainedTypeAtLocation(ctx.TypeChecker, right)) {
				return
			}

			// Source text, not a symbol name — a mangled private name carries a
			// control byte and the diagnostic gets dropped before oxlint sees it.
			r := utils.TrimNodeTextRange(ctx.SourceFile, right)
			ctx.ReportNode(node, buildPrimitiveDefaultMessage(operator, ctx.SourceFile.Text()[r.Pos():r.End()]))
		}

		return rule.RuleListeners{
			ast.KindBinaryExpression: func(node *ast.Node) {
				binary := node.AsBinaryExpression()
				switch binary.OperatorToken.Kind {
				case ast.KindQuestionQuestionToken, ast.KindQuestionQuestionEqualsToken:
					check(node, binary.Right, "??")
				case ast.KindBarBarToken, ast.KindBarBarEqualsToken:
					check(node, binary.Right, "||")
				}
			},
		}
	},
}
