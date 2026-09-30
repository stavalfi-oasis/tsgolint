package no_tracked_close

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

const stub = `
declare class Base {
  track<T>(value: T): T;
  close(): Promise<void>;
  other(): Promise<void>;
}
`

func TestNoTrackedClose(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoTrackedCloseRule, []rule_tester.ValidTestCase{
		{Code: stub + `class A extends Base { async run(): Promise<void> { await this.close(); } }`},
		{Code: stub + `class A extends Base { async run(): Promise<void> { await this.track(this.other()); } }`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   stub + `class A extends Base { async run(): Promise<void> { await this.track(this.close()); } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "trackedClose"}},
			Output: []string{stub + `class A extends Base { async run(): Promise<void> { await this.close(); } }`},
		},
		{
			Code:   stub + `class A extends Base { async run(): Promise<void> { await this.track(super.close()); } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "trackedClose"}},
			Output: []string{stub + `class A extends Base { async run(): Promise<void> { await super.close(); } }`},
		},
	})
}
