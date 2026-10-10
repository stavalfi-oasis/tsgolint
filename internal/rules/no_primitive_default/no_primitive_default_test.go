package no_primitive_default

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoPrimitiveDefault(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoPrimitiveDefaultRule, []rule_tester.ValidTestCase{
		// Containers are the whole point of the exemption.
		{Code: `
declare const value: { a: number } | undefined;
const result = value ?? {};
    `},
		{Code: `
declare const value: number[] | undefined;
const result = value || [];
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
const result = value || null;
    `},
		// `any` carries no information, so reporting it would be guesswork.
		{Code: `
declare const value: string | undefined;
declare const fallback: any;
const result = value ?? fallback;
    `},
		// A mixed union still carries a non-constant branch.
		{Code: `
declare const value: string | undefined;
declare const fallback: "unknown" | { a: number };
const result = value || fallback;
    `},
		// `""` and `0` invent no content.
		{Code: `
declare const value: string | undefined;
const result = value ?? "";
    `},
		{Code: `
declare const value: number | undefined;
const result = value || 0;
    `},
		{Code: `
declare let value: string | undefined;
value ??= "";
    `},
		{Code: `
declare let value: number | undefined;
value ||= 0;
    `},
		{Code: `
declare const value: string | undefined;
const result = value ?? ` + "``" + `;
    `},
		// Not a constant: a runtime `string` is the caller's data, not a guess.
		{Code: `
declare const value: string | undefined;
declare const fallback: string;
const result = value ?? fallback;
    `},
		{Code: `
declare const value: number | undefined;
declare const fallback: number;
const result = value ?? fallback;
    `},
		{Code: `
declare const value: string | undefined;
declare function fallbackName(): string;
const result = value || fallbackName();
    `},
		// A template whose substitution is only known at runtime is not a constant.
		{Code: `
declare const value: string | undefined;
declare const owner: string;
const result = value ?? ` + "`tenant-${owner}`" + `;
    `},
		// `&&` is a different operator.
		{Code: `
declare const value: string;
const result = value && "ready";
    `},
		// `allow` names the values this codebase treats as its own vocabulary.
		{
			Code: `
declare const value: string | undefined;
const result = value ?? "local";
    `,
			Options: NoPrimitiveDefaultOptions{Allow: []any{"local"}},
		},
		{
			Code: `
declare const value: string | undefined;
const OWNER = "local";
const result = value ?? OWNER;
    `,
			Options: NoPrimitiveDefaultOptions{Allow: []any{"local"}},
		},
		// `allow` takes numbers too, for the sentinels a codebase owns.
		{
			Code: `
declare const value: number | undefined;
const result = value ?? -1;
    `,
			Options: NoPrimitiveDefaultOptions{Allow: []any{float64(-1)}},
		},
		{
			Code: `
declare const value: number | undefined;
const result = value || 1;
    `,
			Options: NoPrimitiveDefaultOptions{Allow: []any{float64(1)}},
		},
	}, []rule_tester.InvalidTestCase{
		// `allow` exempts the value it names, not every other constant.
		{
			Code: `
declare const value: string | undefined;
const result = value ?? "unknown";
      `,
			Options: NoPrimitiveDefaultOptions{Allow: []any{"local", float64(-1)}},
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveDefault",
					Line:      3,
					Column:    16,
					EndLine:   3,
					EndColumn: 34,
				},
			},
		},
		// `||` substitutes the same made-up value, and for a wider set of inputs.
		{
			Code: `
declare const value: string | undefined;
const result = value || "unknown";
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveDefault",
					Line:      3,
					Column:    16,
					EndLine:   3,
					EndColumn: 34,
				},
			},
		},
		{
			Code: `
declare const value: number | undefined;
const result = value ?? 1000;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveDefault",
					Line:      3,
					Column:    16,
					EndLine:   3,
					EndColumn: 29,
				},
			},
		},
		{
			Code: `
declare const value: number | undefined;
const result = value ?? -1;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveDefault",
					Line:      3,
					Column:    16,
					EndLine:   3,
					EndColumn: 27,
				},
			},
		},
		// A named constant is still a constant.
		{
			Code: `
declare const value: number | undefined;
const DEFAULT_PORT = 8080;
const result = value ?? DEFAULT_PORT;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveDefault",
					Line:      4,
					Column:    16,
					EndLine:   4,
					EndColumn: 37,
				},
			},
		},
		// Every substitution is known at compile time, so the whole template is.
		{
			Code: `
declare const value: string | undefined;
const owner = "acme";
const result = value ?? ` + "`tenant-${owner}`" + `;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveDefault",
					Line:      4,
					Column:    16,
					EndLine:   4,
					EndColumn: 42,
				},
			},
		},
		// A union of constants is still entirely made up.
		{
			Code: `
declare const value: string | undefined;
declare const fallback: "unknown" | "n/a";
const result = value ?? fallback;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveDefault",
					Line:      4,
					Column:    16,
					EndLine:   4,
					EndColumn: 33,
				},
			},
		},
		// `??=` and `||=` hide the same substitution behind an assignment.
		{
			Code: `
declare let value: string | undefined;
value ??= "unknown";
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveDefault",
					Line:      3,
					Column:    1,
					EndLine:   3,
					EndColumn: 20,
				},
			},
		},
		{
			Code: `
declare let value: number | undefined;
value ||= 1;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveDefault",
					Line:      3,
					Column:    1,
					EndLine:   3,
					EndColumn: 12,
				},
			},
		},
	})
}
