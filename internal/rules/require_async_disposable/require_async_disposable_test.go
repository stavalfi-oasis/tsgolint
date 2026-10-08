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
		// close() no longer has to be last: stopping the workers, closing, then
		// freeing the connection is the order that actually works, because the
		// connection is what the in-flight work close() drains still needs.
		{Code: disposableStub + `
class Service extends ADisposable {
  async [Symbol.asyncDispose](): Promise<void> {
    await this.close();
    await work();
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
		// Two closes: the second one throws at runtime, because close() rejects a
		// second disposal.
		{
			Code: disposableStub + `
class Service extends ADisposable {
  async [Symbol.asyncDispose](): Promise<void> {
    await this.close();
    await work();
    await this.close();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "closeOnce", Line: 12, Column: 11, EndLine: 12, EndColumn: 23},
			},
			Output: []string{disposableStub + `
class Service extends ADisposable {
  async [Symbol.asyncDispose](): Promise<void> {
    await this.close();
    await work();
  }
}
      `},
		},
		// track() inside dispose: the await already waits, and close() clears the
		// set track() feeds, so nothing would ever drain it.
		{
			Code: disposableStub + `
class Service extends ADisposable {
  async [Symbol.asyncDispose](): Promise<void> {
    await this.track(work());
    await this.close();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "noTrack", Line: 10, Column: 11, EndLine: 10, EndColumn: 29},
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
		// The old `await this.track(this.close())` shape: track is reported, and
		// the close inside it still counts as the one required close.
		{
			Code: disposableStub + `
class Service extends ADisposable {
  async [Symbol.asyncDispose](): Promise<void> {
    await this.track(this.close());
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "noTrack", Line: 10, Column: 11, EndLine: 10, EndColumn: 35},
			},
			Output: []string{disposableStub + `
class Service extends ADisposable {
  async [Symbol.asyncDispose](): Promise<void> {
    await this.close();
  }
}
      `},
		},
	})
}
