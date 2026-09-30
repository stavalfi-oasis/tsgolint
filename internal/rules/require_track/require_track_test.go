package require_track

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

func TestRequireTrack(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireTrackRule, []rule_tester.ValidTestCase{
		{Code: disposableStub + `
class Service extends ADisposable {
  async run(): Promise<void> {
    await this.track(work());
  }
}
    `},
		{Code: disposableStub + `
class Service extends ADisposable {
  async run(): Promise<void> {
    await this.close();
  }
}
    `},
		// Storing the promise on the instance is not fire-and-forget: the field's
		// consumer decides whether to track it, and the assignment itself was
		// being mis-read as an untracked call.
		{Code: disposableStub + `
class Service extends ADisposable {
  #drained: Promise<void> | undefined;
  start(): void {
    this.#drained = this.track(work());
  }
}
    `},
		{Code: disposableStub + `
class Service extends ADisposable {
  #drained: Promise<void> | undefined;
  start(): void {
    this.#drained = work();
  }
  async drained(): Promise<void> {
    await this.track(this.#drained);
  }
}
    `},
		// Not an ADisposable, so the rule has nothing to say.
		{Code: disposableStub + `
class Plain {
  async run(): Promise<void> {
    await work();
  }
}
    `},
		// A static member has no instance to track on.
		{Code: disposableStub + `
class Service extends ADisposable {
  static async run(): Promise<void> {
    await work();
  }
}
    `},
		// Not a promise.
		{Code: disposableStub + `
declare function sync(): void;
class Service extends ADisposable {
  run(): void {
    sync();
  }
}
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: disposableStub + `
class Service extends ADisposable {
  async run(): Promise<void> {
    await work();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireTrack", Line: 10, Column: 5, EndLine: 10, EndColumn: 17},
			},
		},
		// The JS rule listened only on AwaitExpression, so an unawaited promise
		// produced no node and went unreported — yet it is exactly the case that
		// escapes the in-flight set.
		{
			Code: disposableStub + `
class Service extends ADisposable {
  run(): void {
    work();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireTrack", Line: 10, Column: 5, EndLine: 10, EndColumn: 12},
			},
		},
		// Extends a subclass of ADisposable. The JS rule compared the superclass
		// identifier to "ADisposable" literally, so this was invisible to it.
		{
			Code: disposableStub + `
class Middle extends ADisposable {}
class Service extends Middle {
  async run(): Promise<void> {
    await work();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireTrack", Line: 11, Column: 5, EndLine: 11, EndColumn: 17},
			},
		},
	})
}
