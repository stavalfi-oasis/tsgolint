package no_private_keyword

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoPrivateKeyword(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoPrivateKeywordRule, []rule_tester.ValidTestCase{
		{Code: `class A { #value = 1; public read(): number { return this.#value; } }`},
		// A parameter property has no `#` spelling, so the constructor is exempt.
		{Code: `class A { public constructor(private readonly value: number) {} }`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `class A { private value = 1; }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "privateKeyword"}},
		},
		{
			Code:   `class A { private read(): void {} }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "privateKeyword"}},
		},
	})
}
