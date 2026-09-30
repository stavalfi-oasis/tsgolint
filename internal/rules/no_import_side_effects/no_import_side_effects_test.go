package no_import_side_effects

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoImportSideEffects(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoImportSideEffectsRule, []rule_tester.ValidTestCase{
		{Code: `export class A { public static run(): void {} }`},
		{Code: `export interface A { value: number }`},
		{Code: `const value = 1; export const doubled = value;`},
		// A call inside a function body only runs when something calls it.
		{Code: `declare function build(): number; export const make = (): number => build();`},
		// Test registration is the file's whole point and cannot move.
		{Code: `declare function describe(name: string, fn: () => void): void; describe("x", () => {});`},
		{Code: `declare function promisify<T>(fn: T): T; declare const raw: () => void; const wrapped = promisify(raw);`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `declare function build(): number; const value = build();`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "runsAtImport"}},
		},
		{
			Code:   `declare class Thing {}; const made = new Thing();`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "runsAtImport"}},
		},
		{
			Code:   `declare function build(): Promise<number>; const value = await build();`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "runsAtImport"}},
		},
		{
			// A bare expression statement at module scope runs, whatever it is.
			Code:   `declare const thing: { run(): void }; thing.run();`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "runsAtImport"}},
		},
		{
			Code:   `import "./foo";`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "sideEffectImport"}},
		},
		{
			Code:   `declare function build(): number; export class A { public static value = build(); }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "staticProperty"}},
		},
	})
}
