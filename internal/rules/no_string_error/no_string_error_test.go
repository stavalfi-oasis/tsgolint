package no_string_error

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoStringError(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoStringErrorRule, []rule_tester.ValidTestCase{
		{Code: `
declare const value: string;
String(value);
    `},
		{Code: `
declare const value: number;
String(value);
    `},
		{Code: `
declare const value: boolean | null | undefined;
String(value);
    `},
		// `any` carries no information, so reporting it would be guesswork.
		{Code: `
declare const value: any;
String(value);
    `},
		// Not the global String call the rule is about.
		{Code: `
declare const error: Error;
String(error, error);
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: `
declare const error: Error;
String(error);
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "stringError",
					Line:      3,
					Column:    1,
					EndLine:   3,
					EndColumn: 14,
				},
			},
		},
		{
			Code: `
declare const value: { a: number };
String(value);
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "stringError",
					Line:      3,
					Column:    1,
					EndLine:   3,
					EndColumn: 14,
				},
			},
		},
		// The name-based JS rule missed this one: the binding is not called
		// `error`, but the value is still an object.
		{
			Code: `
declare const payload: { code: number };
String(payload);
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "stringError",
					Line:      3,
					Column:    1,
					EndLine:   3,
					EndColumn: 16,
				},
			},
		},
		// A union is only safe when every member is.
		{
			Code: `
declare const value: string | Error;
String(value);
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "stringError",
					Line:      3,
					Column:    1,
					EndLine:   3,
					EndColumn: 14,
				},
			},
		},
	})
}
