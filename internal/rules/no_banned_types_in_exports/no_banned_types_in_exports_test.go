package no_banned_types_in_exports

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

var options = NoBannedTypesInExportsOptions{Types: []string{"DeepReadonly"}}

const deepReadonly = "type DeepReadonly<T> = { readonly [K in keyof T]: DeepReadonly<T[K]> };\n"

func TestNoBannedTypesInExports(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoBannedTypesInExportsRule, []rule_tester.ValidTestCase{
		// Not exported.
		{Code: deepReadonly + `type A = DeepReadonly<{ a: string }>;`, Options: options},
		{Code: deepReadonly + `interface A { a: DeepReadonly<{ b: string }> }`, Options: options},
		// Use sites outside type declarations are fine.
		{Code: deepReadonly + `export function f(a: DeepReadonly<{ b: string }>): void {}`, Options: options},
		{Code: deepReadonly + `export class C { public m(a: DeepReadonly<{ b: string }>): void {} }`, Options: options},
		// The banned type's own exported definition.
		{Code: `export type DeepReadonly<T> = { readonly [K in keyof T]: DeepReadonly<T[K]> };`, Options: options},
		// Not in the list.
		{Code: `export type A = Readonly<{ a: string }>;`, Options: options},
	}, []rule_tester.InvalidTestCase{
		{
			Code:    deepReadonly + `export type A = DeepReadonly<{ a: string }>;`,
			Options: options,
			Errors:  []rule_tester.InvalidTestCaseError{{MessageId: "bannedTypeInExport", Line: 2, Column: 17}},
		},
		{
			Code:    deepReadonly + `export interface A { readonly a: readonly DeepReadonly<{ b: string }>[] }`,
			Options: options,
			Errors:  []rule_tester.InvalidTestCaseError{{MessageId: "bannedTypeInExport", Line: 2, Column: 43}},
		},
		{
			Code:    deepReadonly + `export type A = { readonly f: (a: DeepReadonly<{ b: string }>) => void };`,
			Options: options,
			Errors:  []rule_tester.InvalidTestCaseError{{MessageId: "bannedTypeInExport", Line: 2, Column: 35}},
		},
		{
			Code:    deepReadonly + `type DR<T> = DeepReadonly<T>; export type A = DR<{ a: string }>;`,
			Options: NoBannedTypesInExportsOptions{Types: []string{"DR"}},
			Errors:  []rule_tester.InvalidTestCaseError{{MessageId: "bannedTypeInExport", Line: 2, Column: 47}},
		},
	})
}
