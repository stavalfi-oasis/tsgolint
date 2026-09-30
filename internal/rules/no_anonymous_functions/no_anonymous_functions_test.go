package no_anonymous_functions

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoAnonymousFunctions(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoAnonymousFunctionsRule, []rule_tester.ValidTestCase{
		{Code: `function named(): void {}; named();`},
		{Code: `const handler = (): void => {}; handler();`},
		{Code: `[1, 2].map((value) => value * 2);`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `(() => 1)();`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "anonymousIife"}},
		},
		{
			// estree drops the parens, so the JS rule saw a bare function callee.
			Code:   `(function () { return 1; })();`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "anonymousIife"}},
		},
	})
}
