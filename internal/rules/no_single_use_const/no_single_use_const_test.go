package no_single_use_const

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoSingleUseConst(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoSingleUseConstRule, []rule_tester.ValidTestCase{
		{Code: `const label = "x"; export const a = label; export const b = label;`},
		// An exported const is used by files this run may not even see.
		{Code: `export const label = "x"; export const a = label;`},
		// Only literals inline cleanly.
		{Code: `declare function build(): number; const value = build(); export const a = value;`},
		{Code: `const label = "x";`},
		// A const read only through object shorthands is still read: the
		// identifier's own symbol there is the property, not the variable.
		{Code: `const label = "x"; export const a = { label }; export const b = { label };`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `const label = "x"; export const a = label;`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "singleUseConst"}},
		},
		{
			Code:   `const size = 10; export const a = size;`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "singleUseConst"}},
		},
		{
			Code:   "declare const n: number; const label = `x${n}`; export const a = label;",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "singleUseConst"}},
		},
	})
}
