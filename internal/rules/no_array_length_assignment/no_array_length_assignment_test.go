package no_array_length_assignment

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoArrayLengthAssignment(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoArrayLengthAssignmentRule, []rule_tester.ValidTestCase{
		{Code: `
declare const workers: string[];
const count = workers.length;
    `},
		{Code: `
let workers: string[] = [];
workers = [];
    `},
		{Code: `
declare const workers: string[];
if (workers.length === 0) {
}
    `},
		{Code: `
class Pool {
  readonly #workers: string[] = [];
  public clear(): void {
    this.#workers.splice(0, this.#workers.length);
  }
}
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: `
declare const workers: string[];
workers.length = 0;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "lengthAssignment", Line: 3, Column: 1, EndLine: 3, EndColumn: 19},
			},
		},
		{
			Code: `
class Pool {
  readonly #workers: string[] = [];
  public clear(): void {
    this.#workers.length = 0;
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "lengthAssignment", Line: 5, Column: 5, EndLine: 5, EndColumn: 29},
			},
		},
		{
			Code: `
declare const workers: string[];
workers.length -= 1;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "lengthAssignment", Line: 3, Column: 1, EndLine: 3, EndColumn: 20},
			},
		},
		{
			Code: `
declare const workers: string[];
workers['length'] = 0;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "lengthAssignment", Line: 3, Column: 1, EndLine: 3, EndColumn: 22},
			},
		},
	})
}
