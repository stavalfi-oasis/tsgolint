package require_async_disposable

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

const disposableStub = `
declare class ADisposable {
  protected track<T>(value: T): T;
  close(): Promise<void>;
}
declare function work(): Promise<void>;
`

func TestRequireAsyncDisposable(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireAsyncDisposableRule, []rule_tester.ValidTestCase{
		{Code: disposableStub + `
class Service extends ADisposable {
  async [Symbol.asyncDispose](): Promise<void> {
    await this.close();
  }
}
    `},
		{Code: disposableStub + `
class Service extends ADisposable {
  async [Symbol.asyncDispose](): Promise<void> {
    await this.track(this.close());
  }
}
    `},
		// No disposal member at all — a different rule's business.
		{Code: disposableStub + `
class Service extends ADisposable {
  run(): void {}
}
    `},
		// Extends a subclass of ADisposable. The JS rule compared the superclass
		// identifier to "ADisposable" literally, so it wrongly demanded this class
		// extend the base directly.
		{Code: disposableStub + `
class Middle extends ADisposable {}
class Service extends Middle {
  async [Symbol.asyncDispose](): Promise<void> {
    await this.close();
  }
}
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: disposableStub + `
class Service {
  run(): void {}
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "mustExtend", Line: 8, Column: 7, EndLine: 8, EndColumn: 14},
			},
		},
		{
			Code: disposableStub + `
class Service extends ADisposable {
  async [Symbol.asyncDispose](): Promise<void> {
    await work();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "mustClose", Line: 9, Column: 3, EndLine: 11, EndColumn: 4},
			},
			Output: []string{disposableStub + `
class Service extends ADisposable {
  async [Symbol.asyncDispose](): Promise<void> {
    await work();
    await this.close();
  }
}
      `},
		},
		{
			Code: disposableStub + `
class Service extends ADisposable {
  async [Symbol.asyncDispose](): Promise<void> {
    await this.close();
    await work();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "mustCloseLast", Line: 10, Column: 5, EndLine: 10, EndColumn: 24},
			},
			Output: []string{disposableStub + `
class Service extends ADisposable {
  async [Symbol.asyncDispose](): Promise<void> {
    await work();
    await this.close();
  }
}
      `},
		},
	})
}
