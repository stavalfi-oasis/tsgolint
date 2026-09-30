package no_infinite_loop

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoInfiniteLoop(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoInfiniteLoopRule, []rule_tester.ValidTestCase{
		{Code: `declare const signal: { aborted: boolean }; while (!signal.aborted) { break; }`},
		{Code: `for (let i = 0; ; i += 10) { break; }`},
		{Code: `while (false) {}`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `for (;;) { break; }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "infiniteLoop", Line: 1, Column: 1}},
		},
		{
			Code:   `while (true) { break; }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "infiniteLoop", Line: 1, Column: 1}},
		},
	})
}
