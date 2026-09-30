package no_void_promise

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoVoidPromise(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoVoidPromiseRule, []rule_tester.ValidTestCase{
		{Code: `
declare const value: number;
void value;
    `},
		{Code: `
function sync(): void {}
void sync();
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: `
declare const promise: Promise<number>;
void promise;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "voidPromise", Line: 3, Column: 1, EndLine: 3, EndColumn: 13},
			},
		},
		{
			Code: `
async function work(): Promise<void> {}
void work();
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "voidPromise", Line: 3, Column: 1, EndLine: 3, EndColumn: 12},
			},
		},
		// The JS rule guessed promise-ness from the callee name, so an async
		// function reached through an alias was invisible to it.
		{
			Code: `
async function work(): Promise<void> {}
const alias = work;
void alias();
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "voidPromise", Line: 4, Column: 1, EndLine: 4, EndColumn: 13},
			},
		},
		// A plain thenable is still a promise for this rule's purpose.
		{
			Code: `
declare const thenable: { then(onfulfilled: () => void): void };
void thenable;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "voidPromise", Line: 3, Column: 1, EndLine: 3, EndColumn: 14},
			},
		},
	})
}
