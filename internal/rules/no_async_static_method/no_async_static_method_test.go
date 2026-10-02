package no_async_static_method

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoAsyncStaticMethod(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoAsyncStaticMethodRule, []rule_tester.ValidTestCase{
		{Code: `class A { async run(): Promise<void> {} }`},
		{Code: `class A { static run(): void {} }`},
		{Code: `class A { static value = 1; }`},
		{Code: `class A { static make = () => 1; }`},
		{Code: `async function run(): Promise<void> {}`},
		{Code: `const a = { async run() {} };`},
		// The async arrow is nested inside a sync static, not the static itself.
		{Code: `class A { static make(): () => Promise<void> { return async () => {}; } }`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `class A { static async run(): Promise<void> {} }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "asyncStaticMethod"}},
		},
		{
			Code:   `class A { private static async run(): Promise<void> {} }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "asyncStaticMethod"}},
		},
		{
			Code:   `class A { static async *run(): AsyncGenerator<number> { yield 1; } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "asyncStaticMethod"}},
		},
		{
			Code:   `class A { static run = async (): Promise<void> => {}; }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "asyncStaticMethod"}},
		},
		{
			Code:   `class A { static run = async function (): Promise<void> {}; }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "asyncStaticMethod"}},
		},
		{
			Code:   `const A = class { static async run(): Promise<void> {} };`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "asyncStaticMethod"}},
		},
	})
}
