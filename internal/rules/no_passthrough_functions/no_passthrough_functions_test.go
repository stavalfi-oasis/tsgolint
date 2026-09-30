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
		// unicorn/no-array-callback-reference forbids the bare reference an array
		// iterator would otherwise take, so the wrapper arrow there is forced.
		{Code: `
class Service {
  run(values: readonly number[]): number[] {
    return values.map((value) => this.double(value));
  }
  double(value: number): number { return value * 2; }
}
    `},
		{Code: `
declare function matches(file: string): boolean;
declare const files: readonly string[];
export const chosen = files.filter((file) => matches(file));
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
		// A private member's symbol carries TypeScript's mangled internal name
		// ("\x00#1@#sendTwo"). A message holding that control byte is dropped
		// before it reaches oxlint, so the report has to name the call site.
		{
			Code: `
class Uploader {
  wrappedByInheritedTrack(args: { readonly key: string }): Promise<void> {
    return this.track(this.#sendTwo(args));
  }
  #sendTwo(args: { readonly key: string }): Promise<void> {
    const body = JSON.stringify(args);
    return Promise.resolve(body).then(() => undefined);
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "passthroughFunction", Line: 3, Column: 3, EndLine: 3, EndColumn: 26},
			},
		},
	})
}
