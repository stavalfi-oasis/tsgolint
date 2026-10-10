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
		// A mixed union still carries a non-primitive branch.
		{Code: `
declare const value: string | undefined;
declare const fallback: string | { a: number };
const result = value || fallback;
    `},
		// An empty string invents no content, so it is exempt like `[]`.
		{Code: `
declare const value: string | undefined;
const result = value ?? "";
    `},
		{Code: `
declare const value: string | undefined;
const result = value || "";
    `},
		{Code: `
declare let value: string | undefined;
value ??= "";
    `},
		{Code: `
declare let value: string | undefined;
value ||= "";
    `},
		{Code: `
declare const value: string | undefined;
const result = value ?? ` + "``" + `;
    `},
		// `&&` is a different operator.
		{Code: `
declare const value: string;
const result = value && "ready";
    `},
		// A throw keeps the missing case explicit rather than papering over it.
		{Code: `
declare const value: string | undefined;
const result = value ?? (() => { throw new Error("missing"); })();
    `},
	}, []rule_tester.InvalidTestCase{
		// A non-empty string is still a made-up value.
		{
			Code: `
declare const value: string | undefined;
const result = value ?? "unknown";
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
const result = value ?? 0;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveDefault",
					Line:      3,
					Column:    16,
					EndLine:   3,
					EndColumn: 26,
				},
			},
		},
		// `||` substitutes the same made-up value, and for a wider set of inputs.
		{
			Code: `
declare const value: number | undefined;
const result = value || 0;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveDefault",
					Line:      3,
					Column:    16,
					EndLine:   3,
					EndColumn: 26,
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
		// Not a literal — the rule reads the type, so a named constant is caught too.
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
		// A call returning a string is just as made-up a default.
		{
			Code: `
declare const value: string | undefined;
declare function fallbackName(): string;
const result = value || fallbackName();
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{
					MessageId: "primitiveDefault",
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
					MessageId: "primitiveDefault",
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
