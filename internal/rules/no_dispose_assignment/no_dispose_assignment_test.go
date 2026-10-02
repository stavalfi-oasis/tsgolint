package no_dispose_assignment

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoDisposeAssignment(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoDisposeAssignmentRule, []rule_tester.ValidTestCase{
		{Code: `
class Pool {
  #open = true;
  public stop(): void {
    this.#open = false;
  }
}
    `},
		{Code: `
class Pool {
  readonly #workers: string[] = [];
  public async [Symbol.asyncDispose](): Promise<void> {
    const workers = this.#workers;
    for (const worker of workers) {
      worker.trim();
    }
  }
}
    `},
		{Code: `
class Pool {
  public [Symbol.dispose](): void {
    const other = { open: true };
    other.open = false;
  }
}
    `},
		{Code: `
class Pool {
  #open = true;
  public [Symbol.dispose](): void {
    function reset(this: { open: boolean }): void {
      this.open = false;
    }
    reset.call({ open: true });
  }
}
    `},
		{Code: `
class Pool {
  public [Symbol.dispose](): void {
    class Inner {
      #open = true;
      public close(): void {
        this.#open = false;
      }
    }
    new Inner().close();
  }
}
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: `
class Pool {
  #workers: string[] = [];
  #connection: string | undefined = undefined;
  public async [Symbol.asyncDispose](): Promise<void> {
    this.#workers.length = 0;
    this.#connection = undefined;
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "disposeAssignment", Line: 6, Column: 5, EndLine: 6, EndColumn: 25},
				{MessageId: "disposeAssignment", Line: 7, Column: 5, EndLine: 7, EndColumn: 21},
			},
		},
		{
			Code: `
class Pool {
  #open = true;
  public [Symbol.dispose](): void {
    this.#open = false;
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "disposeAssignment", Line: 5, Column: 5, EndLine: 5, EndColumn: 15},
			},
		},
		{
			Code: `
class Pool {
  #count = 0;
  public [Symbol.dispose](): void {
    [1, 2].forEach(() => {
      this.#count++;
    });
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "disposeAssignment", Line: 6, Column: 7, EndLine: 6, EndColumn: 18},
			},
		},
		{
			Code: `
class Pool {
  public cache: Record<string, string> = {};
  public [Symbol.dispose](): void {
    delete this.cache['a'];
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "disposeAssignment", Line: 5, Column: 12, EndLine: 5, EndColumn: 27},
			},
		},
		{
			Code: `
class Pool {
  public a = '';
  public b = '';
  public [Symbol.dispose](): void {
    ({ a: this.a, b: this.b } = { a: '', b: '' });
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "disposeAssignment", Line: 6, Column: 11, EndLine: 6, EndColumn: 17},
				{MessageId: "disposeAssignment", Line: 6, Column: 22, EndLine: 6, EndColumn: 28},
			},
		},
		{
			Code: `
class Pool {
  #count = 0;
  public async [Symbol.asyncDispose](): Promise<void> {
    this.#count += 1;
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "disposeAssignment", Line: 5, Column: 5, EndLine: 5, EndColumn: 16},
			},
		},
	})
}
