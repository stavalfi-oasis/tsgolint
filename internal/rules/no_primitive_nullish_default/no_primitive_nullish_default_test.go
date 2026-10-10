package no_primitive_nullish_default

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoPrimitiveNullishDefault(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoPrimitiveNullishDefaultRule, []rule_tester.ValidTestCase{
		// Containers are the whole point of the exemption.
		{Code: `
declare const value: { a: number } | undefined;
const result = value ?? {};
    `},
		{Code: `
declare const value: number[] | undefined;
const result = value ?? [];
    `},
		// Booleans, null and undefined are not strings or numbers.
		{Code: `
declare const value: boolean | undefined;
const result = value ?? false;
    `},
		{Code: `
declare const value: string | undefined;
const result = value ?? undefined;
    `},
		{Code: `
declare const value: string | undefined;
const result = value ?? null;
    `},
		// `any` carries no information, so reporting it would be guesswork.
		{Code: `
declare const value: string | undefined;
declare const fallback: any;
const result = value ?? fallback;
    `},
		// A mixed union still carries a non-primitive branch.
		{Code: `
declare const value: string | undefined;
declare const fallback: string | { a: number };
const result = value ?? fallback;
    `},
		// `||` and `&&` are a different operator.
		{Code: `
declare const value: string;
const result = value || "";
    `},
		// A throw keeps the nullish case explicit rather than papering over it.
		{Code: `
declare const value: string | undefined;
const result = value ?? (() => { throw new Error("missing"); })();
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: `
declare const value: string | undefined;
const result = value ?? "";
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveNullishDefault",
					Line:      3,
					Column:    16,
					EndLine:   3,
					EndColumn: 27,
				},
			},
		},
		{
			Code: `
declare const value: number | undefined;
const result = value ?? 0;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveNullishDefault",
					Line:      3,
					Column:    16,
					EndLine:   3,
					EndColumn: 26,
				},
			},
		},
		// Not a literal — the rule reads the type, so a named constant is caught too.
		{
			Code: `
declare const value: number | undefined;
const DEFAULT_PORT = 8080;
const result = value ?? DEFAULT_PORT;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveNullishDefault",
					Line:      4,
					Column:    16,
					EndLine:   4,
					EndColumn: 37,
				},
			},
		},
		// A call returning a string is just as made-up a default.
		{
			Code: `
declare const value: string | undefined;
declare function fallbackName(): string;
const result = value ?? fallbackName();
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveNullishDefault",
					Line:      4,
					Column:    16,
					EndLine:   4,
					EndColumn: 39,
				},
			},
		},
		// Template literal with a substitution is still a string.
		{
			Code: `
declare const value: string | undefined;
declare const owner: string;
const result = value ?? ` + "`tenant-${owner}`" + `;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveNullishDefault",
					Line:      4,
					Column:    16,
					EndLine:   4,
					EndColumn: 42,
				},
			},
		},
		// A string|number union is still entirely string/number.
		{
			Code: `
declare const value: string | undefined;
declare const fallback: string | number;
const result = value ?? fallback;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveNullishDefault",
					Line:      4,
					Column:    16,
					EndLine:   4,
					EndColumn: 33,
				},
			},
		},
		// `??=` hides the same substitution behind an assignment.
		{
			Code: `
declare let value: string | undefined;
value ??= "";
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveNullishDefault",
					Line:      3,
					Column:    1,
					EndLine:   3,
					EndColumn: 13,
				},
			},
		},
	})
}
