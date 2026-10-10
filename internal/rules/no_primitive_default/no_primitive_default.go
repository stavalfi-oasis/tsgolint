package no_primitive_default

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// Only string and number. A boolean, an object literal, an array literal or a
// call returning a fresh container is a real fallback value; a string or a
// number is a made-up one.
const primitiveDefaultFlags = checker.TypeFlagsStringLike | checker.TypeFlagsNumberLike

func buildPrimitiveDefaultMessage(operator string, text string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "primitiveDefault",
		Description: "`" + operator + " " + text + "` substitutes a made-up string/number for a missing value.",
		Help:        "Let the missing value through and handle it where the meaning of \"missing\" is known, or make the source non-nullable. `{}`, `[]` and `\"\"` are fine.",
	}
}

var NoPrimitiveDefaultRule = rule.Rule{
	Name: "no-primitive-default",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		// A union only counts when every member is a string or a number —
		// `string | Config` still carries the object branch, and `any` carries
		// no information at all, so reporting it would be guesswork.
		isPrimitiveDefault := func(t *checker.Type) bool {
			return !utils.SomeUnionTypePart(t, func(part *checker.Type) bool {
				return !utils.IsTypeFlagSet(part, primitiveDefaultFlags)
			})
		}

		// `?? ""` is the one exemption: an empty string invents no content, it
		// is the identity of the string being built, the same way `?? []` is.
		// The flag check guards the cast — `AsLiteralType` panics on `string`.
		isEmptyString := func(t *checker.Type) bool {
			return !utils.SomeUnionTypePart(t, func(part *checker.Type) bool {
				if !utils.IsTypeFlagSet(part, checker.TypeFlagsStringLiteral) {
					return true
				}
				literal := part.AsLiteralType()
				return literal == nil || literal.Value() != ""
			})
		}

		check := func(node *ast.Node, right *ast.Node, operator string) {
			t := utils.GetConstrainedTypeAtLocation(ctx.TypeChecker, right)
			if !isPrimitiveDefault(t) || isEmptyString(t) {
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
