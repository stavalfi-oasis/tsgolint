package no_object_destructuring

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoObjectDestructuring(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoObjectDestructuringRule, []rule_tester.ValidTestCase{
		{Code: `declare function newTenant(): { id: string }; const { id } = newTenant();`},
		{Code: `declare function newTenant(): Promise<{ id: string }>; async function f() { const { id } = await newTenant(); }`},
		{Code: `declare function load(): { data: { id: string } }; const { id } = load().data;`},
		{Code: `class C { m() { return { id: "" } } } const { id } = new C().m();`},
		{Code: `declare const a: { id: string }; declare const b: { id: string }; const { id } = a ?? b;`},
		{Code: `declare const c: boolean; declare const a: { id: string }; declare const b: { id: string }; const { id } = c ? a : b;`},
		{Code: `declare const rows: readonly { id: string }[]; for (const { id } of rows) { console.log(id); }`},
		{Code: `function f({ id }: { id: string }): void { console.log(id); }`},
		{Code: `declare function g(cb: (arg: { id: string }) => void): void; g(({ id }) => console.log(id));`},
		{Code: `function f({ nested: { id } }: { nested: { id: string } }): void { console.log(id); }`},
		{Code: `declare const helper: { etl: { id: string } }; const [{ id }] = [helper.etl];`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `declare const testHelper: { etl: { id: string } }; const { etl } = testHelper;`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "objectDestructuring", Line: 1, Column: 58}},
		},
		{
			Code:   `declare const testHelper: { etl: { repo: string } }; const { repo } = testHelper.etl;`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "objectDestructuring", Line: 1, Column: 60}},
		},
		{
			Code:   `class C { #logger = { name: "" }; m(): void { const { name } = this.#logger; console.log(name); } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "objectDestructuring", Line: 1, Column: 53}},
		},
		{
			Code:   `declare const testHelper: { etl: { id: string } } | undefined; const { etl } = testHelper!;`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "objectDestructuring", Line: 1, Column: 70}},
		},
		{
			Code:   `declare const testHelper: unknown; const { etl } = testHelper as { etl: string };`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "objectDestructuring", Line: 1, Column: 42}},
		},
		{
			Code:   `declare const helpers: readonly { etl: string }[]; const { etl } = helpers[0];`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "objectDestructuring", Line: 1, Column: 58}},
		},
		{
			// A nested pattern rides on its root, so one report covers the statement.
			Code:   `declare const testHelper: { etl: { repo: string } }; const { etl: { repo } } = testHelper;`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "objectDestructuring", Line: 1, Column: 60}},
		},
		{
			Code:   `try {} catch ({ message }) { console.log(message); }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "objectDestructuring", Line: 1, Column: 15}},
		},
		{
			Code:   `declare const b: { a: string }; let a: string; ({ a } = b);`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "objectDestructuring", Line: 1, Column: 49}},
		},
		{
			// Banned even when a call produced the value, because the form itself is.
			Code:   `declare function load(): { a: string }; let a: string; ({ a } = load());`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "objectDestructuring", Line: 1, Column: 57}},
		},
	})
}
