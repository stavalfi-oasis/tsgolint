package require_object_params

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestRequireObjectParams(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireObjectParamsRule, []rule_tester.ValidTestCase{
		{Code: `function one({ a, b }: { a: number; b: number }): number { return a + b; }`},
		{Code: `function one(a: number): number { return a; }`},
		// A `this` parameter is a type annotation, not an argument.
		{Code: `function one(this: void, a: number): number { return a; }`},
		{Code: `class A { public get value(): number { return 1; } public set value(next: number) {} }`},
		// The JS rule only reached an arrow through a variable declarator.
		{Code: `declare function run(fn: (a: number, b: number) => void): void; run((a, b) => {});`},
		{Code: `async function one({ signal }: { readonly signal: AbortSignal }): Promise<void> {}`},
		// Any required property typed AbortSignal counts, whatever its name.
		{Code: `const one = async ({ activityAbortSignal }: { readonly activityAbortSignal: AbortSignal; readonly n: number }): Promise<number> => 1;`},
		{Code: `class A { public run({ signal }: { readonly signal: AbortSignal }): Promise<number> { return Promise.resolve(1); } }`},
		// A constructor, an override, a symbol-named method and an object-literal method have a dictated shape.
		{Code: `class A { public constructor() {} }`},
		{Code: `declare class B { public run(): Promise<void>; } class A extends B { public override async run(): Promise<void> {} }`},
		{Code: `class A { public async [Symbol.asyncDispose](): Promise<void> {} }`},
		{Code: `const handler = { async fetch(): Promise<void> {} };`},
		{
			Code: `import type { Driver } from "./driver"; class A implements Driver { public async init(): Promise<void> {} }`,
			Files: map[string]string{"driver.d.ts": `export interface Driver { init(): Promise<void>; }`},
		},
		{
			Code: `import { Base } from "./base"; class A extends Base { public async lock(): Promise<void> {} }`,
			Files: map[string]string{"base.d.ts": `export declare abstract class Base { abstract lock(): Promise<void>; }`},
		},
		// Overloads and abstract members have no body to cancel.
		{Code: `abstract class A { public abstract run(): Promise<void>; }`},
		{Code: `declare function run(): Promise<void>;`},
		{Code: `function one(): number { return 1; }`},
		// An inline callback is not checked.
		{Code: `declare function run(fn: () => Promise<void>): void; run(async () => {});`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `function two(a: number, b: number): number { return a + b; }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "tooManyParams"}},
		},
		{
			Code:   `const two = (a: number, b: number): number => a + b;`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "tooManyParams"}},
		},
		{
			Code:   `class A { public two(a: number, b: number): number { return a + b; } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "tooManyParams"}},
		},
		{
			Code:   `class A { public constructor(a: number, b: number) {} }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "tooManyParams"}},
		},
		{
			Code:   `async function none(): Promise<void> {}`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingSignal"}},
		},
		{
			Code:   `const noSignal = async ({ n }: { readonly n: number }): Promise<number> => n;`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingSignal"}},
		},
		{
			Code:   `function promised(): Promise<number> { return Promise.resolve(1); }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingSignal"}},
		},
		{
			Code:   `class A { public async run({ signal }: { readonly signal?: AbortSignal }): Promise<void> {} }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingSignal"}},
		},
		{
			Code:   `class A { async #run(n: number): Promise<number> { return n; } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingSignal"}},
		},
		{
			Code:   `interface Ours { init(): Promise<void>; } class A implements Ours { public async init(): Promise<void> {} }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingSignal"}},
		},
	})
}
