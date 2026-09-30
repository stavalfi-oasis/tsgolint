package no_passthrough_functions

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoPassthroughFunctions(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoPassthroughFunctionsRule, []rule_tester.ValidTestCase{
		// Does real work, not forwarding.
		{Code: `
function target(value: number): number { return value; }
function wrapper(value: number): number { return target(value + 1); }
    `},
		// More than one statement.
		{Code: `
function target(value: number): number { return value; }
function wrapper(value: number): number {
  const doubled = value * 2;
  return target(doubled);
}
    `},
		// The target is imported, so the wrapper is the local seam. Resolving the
		// symbol is what tells these apart.
		{Code: `
import { target } from "./foo";
export function wrapper(value: number): number { return target(value); }
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: `
function target(value: number): number { return value; }
function wrapper(value: number): number { return target(value); }
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "passthroughFunction", Line: 3, Column: 10, EndLine: 3, EndColumn: 17},
			},
		},
		{
			Code: `
class Service {
  target(value: number): number { return value; }
  wrapper(value: number): number { return this.target(value); }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "passthroughFunction", Line: 4, Column: 3, EndLine: 4, EndColumn: 10},
			},
		},
		// await + spread still forwards nothing of its own.
		{
			Code: `
async function target(...values: number[]): Promise<number> { return values.length; }
async function wrapper(...values: number[]): Promise<number> { return await target(...values); }
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "passthroughFunction", Line: 3, Column: 16, EndLine: 3, EndColumn: 23},
			},
		},
	})
}
