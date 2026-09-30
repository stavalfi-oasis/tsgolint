package no_string_raw

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoStringRaw(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoStringRawRule, []rule_tester.ValidTestCase{
		{Code: "const pattern = `\\\\d+`;"},
		{Code: "declare function tag(parts: TemplateStringsArray): string; const value = tag`\\d+`;"},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   "const pattern = String.raw`\\d+`;",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "stringRaw"}},
			Output: []string{"const pattern = `\\\\d+`;"},
		},
		{
			Code:   "declare const n: number; const pattern = String.raw`\\d${n}\\w`;",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "stringRaw"}},
			Output: []string{"declare const n: number; const pattern = `\\\\d${n}\\\\w`;"},
		},
	})
}
