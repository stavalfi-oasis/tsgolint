package no_async_global_functions

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoAsyncGlobalFunctions(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoAsyncGlobalFunctionsRule, []rule_tester.ValidTestCase{
		// A global that neither returns a promise nor takes a callback owns no
		// pending work, so there is nothing to dispose.
		{Code: `JSON.stringify({ a: 1 });`},
		{Code: `structuredClone({ a: 1 });`},
		{Code: `
const value = parseInt("1", 10);
    `},
		// `JSON.stringify` accepts a replacer, but this call does not pass one.
		// Keying on the declared parameters instead of the arguments would
		// report every call to it.
		{Code: `
const replacer = (key: string, value: unknown): unknown => value;
JSON.stringify({ a: 1 }, undefined, 2);
    `},
		// The repo's own code is already owned by whoever declared it.
		{Code: `
async function load(): Promise<void> {}
load();
    `},
		{Code: `
const numbers = [1, 2, 3];
numbers.map((n) => n * 2);
    `},
		{Code: `
declare const storage: { read(): Promise<string> };
storage.read();
    `},
		// A method call has no root identifier to resolve to a global.
		{Code: `
class Service {
  private async start(): Promise<void> {}
  public run(): void {
    void this.start();
  }
}
    `},
		// excludeFunctions, by bare name and by the whole callee text.
		{
			Code:    `setTimeout(() => {}, 1000);`,
			Options: NoAsyncGlobalFunctionsOptions{ExcludeFunctions: []string{"setTimeout"}},
		},
		{
			Code:    `Promise.resolve(1);`,
			Options: NoAsyncGlobalFunctionsOptions{ExcludeFunctions: []string{"Promise.resolve"}},
		},
		// excludeModules exempts every export of the module at once.
		{
			Code: `
import { load, work } from 'fake-async';
void load();
work(() => {});
    `,
			Options: NoAsyncGlobalFunctionsOptions{ExcludeModules: []string{"fake-async"}},
		},
		// Excluding one module says nothing about the globals around it.
		{
			Code: `
import { load } from 'fake-async';
void load();
    `,
			Options: NoAsyncGlobalFunctionsOptions{ExcludeModules: []string{"fake-async", "node:fs/promises"}},
		},
	}, []rule_tester.InvalidTestCase{
		{
			Code: `setTimeout(() => {}, 1000);`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "asyncGlobalFunction", Line: 1, Column: 1, EndColumn: 11},
			},
		},
		{
			Code: `queueMicrotask(() => {});`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "asyncGlobalFunction", Line: 1, Column: 1, EndColumn: 15},
			},
		},
		{
			Code: `void fetch('https://example.com');`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "asyncGlobalFunction", Line: 1, Column: 6, EndColumn: 11},
			},
		},
		{
			Code: `void Promise.resolve(1);`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "asyncGlobalFunction", Line: 1, Column: 6, EndColumn: 21},
			},
		},
		// An imported function is reported the same as a bare global: the
		// binding is local, its declaration is not.
		{
			Code: `
import { load } from 'fake-async';
void load();
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "asyncGlobalFunction", Line: 3, Column: 6, EndColumn: 10},
			},
		},
		{
			Code: `
import { work } from 'fake-async';
work(() => {});
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "asyncGlobalFunction", Line: 3, Column: 1, EndColumn: 5},
			},
		},
		// A namespace import resolves through its root identifier.
		{
			Code: `
import * as fakeAsync from 'fake-async';
fakeAsync.work(() => {});
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "asyncGlobalFunction", Line: 3, Column: 1, EndColumn: 15},
			},
		},
		// Excluding a different module leaves this one reported.
		{
			Code: `
import { load } from 'fake-async';
void load();
      `,
			Options: NoAsyncGlobalFunctionsOptions{ExcludeModules: []string{"node:fs/promises"}},
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "asyncGlobalFunction", Line: 3, Column: 6, EndColumn: 10},
			},
		},
	})
}
