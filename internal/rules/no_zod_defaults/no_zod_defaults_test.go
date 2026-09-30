package no_zod_defaults

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

// zod's schema classes are named ZodSomething, which is what the rule keys off.
const zodStub = `
declare class ZodString {
  default(value: string): ZodString;
  prefault(value: string): ZodString;
  optional(): ZodString;
}
declare const z: { string(): ZodString };
`

func TestNoZodDefaults(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoZodDefaultsRule, []rule_tester.ValidTestCase{
		{Code: zodStub + `z.string().optional();`},
		// Same method name, not a zod schema — the JS rule's `z` root check also
		// passed here, but only by accident of the binding name.
		{Code: `
declare const settings: { default(value: string): void };
settings.default("x");
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: zodStub + `z.string().default("x");`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "zodDefault", Line: 8, Column: 1, EndLine: 8, EndColumn: 24},
			},
		},
		{
			Code: zodStub + `z.string().prefault("x");`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "zodDefault", Line: 8, Column: 1, EndLine: 8, EndColumn: 25},
			},
		},
		// The JS rule walked back to an identifier literally named `z`, so a
		// schema held in any other binding slipped through.
		{
			Code: zodStub + `
const schema = z.string();
schema.default("x");
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "zodDefault", Line: 10, Column: 1, EndLine: 10, EndColumn: 20},
			},
		},
	})
}
