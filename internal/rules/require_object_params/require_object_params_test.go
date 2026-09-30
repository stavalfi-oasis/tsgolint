package require_object_params

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestRequireObjectParams(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireObjectParamsRule, []rule_tester.ValidTestCase{
		{Code: `function one({ a, b }: { a: number; b: number }): number { return a + b; }`},
		{Code: `function one(a: number): number { return a; }`},
		// A `this` parameter is a type annotation, not an argument.
		{Code: `function one(this: void, a: number): number { return a; }`},
		{Code: `class A { public get value(): number { return 1; } public set value(next: number) {} }`},
		// The JS rule only reached an arrow through a variable declarator.
		{Code: `declare function run(fn: (a: number, b: number) => void): void; run((a, b) => {});`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `function two(a: number, b: number): number { return a + b; }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "tooManyParams"}},
		},
		{
			Code:   `const two = (a: number, b: number): number => a + b;`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "tooManyParams"}},
		},
		{
			Code:   `class A { public two(a: number, b: number): number { return a + b; } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "tooManyParams"}},
		},
		{
			Code:   `class A { public constructor(a: number, b: number) {} }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "tooManyParams"}},
		},
	})
}
