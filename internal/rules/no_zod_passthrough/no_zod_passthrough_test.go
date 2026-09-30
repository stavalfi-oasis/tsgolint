package no_zod_passthrough

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

const zodStub = `
declare class ZodObject {
  passthrough(): ZodObject;
  loose(): ZodObject;
}
declare const z: { object(): ZodObject };
`

func TestNoZodPassthrough(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoZodPassthroughRule, []rule_tester.ValidTestCase{
		{Code: zodStub + `z.object().loose();`},
		// The JS rule reported any zero-argument `.passthrough()`, so this
		// unrelated stream API was a false positive.
		{Code: `
declare const stream: { passthrough(): void };
stream.passthrough();
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: zodStub + `z.object().passthrough();`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "zodPassthrough", Line: 7, Column: 1, EndLine: 7, EndColumn: 25},
			},
		},
	})
}
