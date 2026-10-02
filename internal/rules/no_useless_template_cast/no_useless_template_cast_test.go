package no_useless_template_cast

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoUselessTemplateCast(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoUselessTemplateCastRule, []rule_tester.ValidTestCase{
		// Outside a template literal the cast is the only thing producing a string.
		{Code: `
declare const index: number;
const id = String(index);
    `},
		// `${sym}` throws, so the cast is load-bearing.
		{Code: `
declare const value: symbol;
const id = ` + "`it-${String(value)}`" + `;
    `},
		{Code: `
declare const value: number | symbol;
const id = ` + "`it-${String(value)}`" + `;
    `},
		// Objects are no-string-error's business, not this rule's.
		{Code: `
declare const value: { a: number };
const id = ` + "`it-${String(value)}`" + `;
    `},
		// `any` carries no information, so reporting it would be guesswork.
		{Code: `
declare const value: any;
const id = ` + "`it-${String(value)}`" + `;
    `},
		// A radix changes the output.
		{Code: `
declare const index: number;
const id = ` + "`it-${index.toString(2)}`" + `;
    `},
		// `x?.toString()` yields undefined, not "undefined".
		{Code: `
declare const index: number | undefined;
const id = ` + "`it-${index?.toString()}`" + `;
    `},
		// Nullish receivers throw on `.toString()`, so the forms are not equivalent.
		{Code: `
declare const value: number | null;
const id = ` + "`it-${value.toString()}`" + `;
    `},
		// A tag function receives the raw values, so no conversion happens.
		{Code: `
declare function tag(strings: TemplateStringsArray, ...values: string[]): string;
declare const index: number;
const id = tag` + "`it-${String(index)}`" + `;
    `},
		// Not the global String call the rule is about.
		{Code: `
declare const index: number;
const id = ` + "`it-${String(index, index)}`" + `;
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: `
declare const index: number;
const id = ` + "`it-${String(index)}`" + `;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "uselessStringCall",
					Line:      3,
					Column:    18,
					EndLine:   3,
					EndColumn: 31,
				},
			},
		},
		{
			Code: `
declare const flag: boolean;
const id = ` + "`it-${String(flag)}`" + `;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "uselessStringCall",
					Line:      3,
					Column:    18,
					EndLine:   3,
					EndColumn: 30,
				},
			},
		},
		// A union is redundant only when every member is — this one is.
		{
			Code: `
declare const value: string | number | null | undefined;
const id = ` + "`it-${String(value)}`" + `;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "uselessStringCall",
					Line:      3,
					Column:    18,
					EndLine:   3,
					EndColumn: 31,
				},
			},
		},
		{
			Code: `
declare const index: number;
const id = ` + "`it-${index.toString()}`" + `;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "uselessToString",
					Line:      3,
					Column:    18,
					EndLine:   3,
					EndColumn: 34,
				},
			},
		},
		// Every span is checked, not just the first.
		{
			Code: `
declare const a: number;
declare const b: bigint;
const id = ` + "`${a}-${String(b)}`" + `;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "uselessStringCall",
					Line:      4,
					Column:    20,
					EndLine:   4,
					EndColumn: 29,
				},
			},
		},
	})
}
