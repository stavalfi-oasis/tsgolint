package require_zod_describe

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

const zodStub = `
declare class ZodString {
  min(n: number): ZodString;
  describe(text: string): ZodString;
}
declare class ZodObject {
  strict(): ZodObject;
  describe(text: string): ZodObject;
}
declare const z: {
  string(): ZodString;
  object(shape: Record<string, ZodString>): ZodObject;
  compile<T>(schema: T): T;
};
`

func TestRequireZodDescribe(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireZodDescribeRule, []rule_tester.ValidTestCase{
		{Code: zodStub + `const schema = z.compile(z.string().describe("name"));`},
		// describe() anywhere along the root chain counts.
		{Code: zodStub + `const schema = z.compile(z.string().describe("name").min(1));`},
		// Only the root needs a description, not its members.
		{Code: zodStub + `const schema = z.compile(z.object({ a: z.string() }).describe("thing"));`},
		// compile() on something that is not the zod namespace.
		{Code: `
declare const re: { compile(source: string): void };
re.compile("x");
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: zodStub + `const schema = z.compile(z.string());`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireZodDescribe", Line: 15, Column: 26, EndLine: 15, EndColumn: 36},
			},
		},
		// A described member does not describe the root.
		{
			Code: zodStub + `const schema = z.compile(z.object({ a: z.string().describe("a") }).strict());`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireZodDescribe", Line: 15, Column: 26, EndLine: 15, EndColumn: 76},
			},
		},
	})
}
