package async_dispose_last

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestAsyncDisposeLast(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &AsyncDisposeLastRule, []rule_tester.ValidTestCase{
		{Code: `class A { public run(): void {} public async [Symbol.asyncDispose](): Promise<void> {} }`},
		{Code: `class A { public run(): void {} }`},
	}, []rule_tester.InvalidTestCase{
		{
			Code: `class A {
  public async [Symbol.asyncDispose](): Promise<void> {}

  public run(): void {}
}`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "asyncDisposeNotLast"}},
			Output: []string{`class A {
  public run(): void {}

  public async [Symbol.asyncDispose](): Promise<void> {}
}`},
		},
	})
}
