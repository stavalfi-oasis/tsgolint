package require_type_annotation

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestRequireTypeAnnotation(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireTypeAnnotationRule, []rule_tester.ValidTestCase{
		{Code: `class A { public readonly app: number; public constructor() { this.app = 1; } }`},
		{Code: `class A { public readonly app = 1; }`},
		{Code: `class A { readonly #app: string; public constructor() { this.#app = ""; } }`},
		{Code: `declare class A { public readonly app: number; }`},
		{Code: `const value = 1;`},
		{Code: `let value: number;`},
		{Code: `const value: number = 1;`},
		{Code: `for (const entry of [1, 2]) { console.log(entry); }`},
		{Code: `for (const key in { a: 1 }) { console.log(key); }`},
		{Code: `const { a, b } = { a: 1, b: 2 };`},
		{Code: `try { throw new Error("x"); } catch (error) { console.log(error); }`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `class A { public readonly app; public constructor() { this.app = 1; } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingTypeAnnotation"}},
		},
		{
			Code:   `class A { #app; public constructor() { this.#app = 1; } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingTypeAnnotation"}},
		},
		{
			Code:   `let value;`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingTypeAnnotation"}},
		},
		{
			Code:   `let first, second: number;`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingTypeAnnotation"}},
		},
	})
}
