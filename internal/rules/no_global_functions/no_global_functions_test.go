package no_global_functions

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoGlobalFunctions(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoGlobalFunctionsRule, []rule_tester.ValidTestCase{
		{Code: `class A { public static run(): void {} }`},
		// Nested functions are a local detail; only module scope is banned.
		{Code: `class A { public static run(): void { const inner = (): void => {}; inner(); } }`},
		{Code: `declare function ambient(): void;`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `function run(): void {}`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "globalFunction"}},
		},
		{
			Code:   `export function run(): void {}`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "globalFunction"}},
		},
		{
			Code:   `const run = (): void => {};`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "globalFunction"}},
		},
		{
			Code:   `export const run = function (): void {};`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "globalFunction"}},
		},
	})
}
