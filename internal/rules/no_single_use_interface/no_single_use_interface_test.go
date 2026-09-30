package no_single_use_interface

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoSingleUseInterface(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoSingleUseInterfaceRule, []rule_tester.ValidTestCase{
		{Code: `
interface Options { port: number }
declare const a: Options;
declare const b: Options;
    `},
		// Exported, so its other uses are in other files.
		{Code: `
export interface Options { port: number }
declare const a: Options;
    `},
		// Never used — a different rule's business.
		{Code: `
interface Options { port: number }
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: `
interface Options { port: number }
declare const a: Options;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "singleUseInterface", Line: 2, Column: 11, EndLine: 2, EndColumn: 18},
			},
		},
	})
}
