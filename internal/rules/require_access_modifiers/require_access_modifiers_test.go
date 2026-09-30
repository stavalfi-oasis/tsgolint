package require_access_modifiers

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestRequireAccessModifiers(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireAccessModifiersRule, []rule_tester.ValidTestCase{
		{Code: `class A { public value = 1; }`},
		// `#field` is already private; an access modifier on it is a syntax error.
		{Code: `class A { #value = 1; }`},
		{Code: `class A { public constructor() {} }`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `class A { value = 1; }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingAccessModifier"}},
			Output: []string{`class A { public value = 1; }`},
		},
		{
			Code:   `class A { read(): void {} }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingAccessModifier"}},
			Output: []string{`class A { public read(): void {} }`},
		},
		{
			Code:   `class A { static read(): void {} }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingAccessModifier"}},
			Output: []string{`class A { public static read(): void {} }`},
		},
	})
}
