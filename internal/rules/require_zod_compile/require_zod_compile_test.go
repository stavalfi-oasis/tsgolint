package require_zod_compile

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

const zodStub = `
declare class ZodString {
  min(n: number): ZodString;
  parse(value: unknown): string;
}
declare const z: {
  string(): ZodString;
  compile<T>(schema: T): T;
};
`

func TestRequireZodCompile(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireZodCompileRule, []rule_tester.ValidTestCase{
		{Code: zodStub + `const schema = z.compile(z.string());`},
		{Code: zodStub + `const schema = z.compile(z.string().min(1));`},
		// A terminal call consumes a schema rather than producing one.
		{Code: zodStub + `
declare const compiled: ZodString;
const value = compiled.parse("x");
    `},
		{Code: `const port = 8080;`},
	}, []rule_tester.InvalidTestCase{
		{
			Code: zodStub + `const schema = z.string();`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireZodCompile", Line: 10, Column: 16, EndLine: 10, EndColumn: 26},
			},
			Output: []string{zodStub + `const schema = z.compile(z.string());`},
		},
		// Reported once, on the outermost expression — not on every link in the
		// chain.
		{
			Code: zodStub + `const schema = z.string().min(1);`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireZodCompile", Line: 10, Column: 16, EndLine: 10, EndColumn: 33},
			},
			Output: []string{zodStub + `const schema = z.compile(z.string().min(1));`},
		},
	})
}

func TestRequireZodCompileHelperReturn(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireZodCompileRule, []rule_tester.ValidTestCase{
		// A helper that already returns z.compile(...) is not a schema the call
		// site can wrap. Reporting here produced a fix that inserted
		// "Storage.compile(" and corrupted two poc files.
		{Code: zodStub + `
class Storage {
  static schema(): ZodString { return z.compile(z.string()); }
  static read(value: unknown): string { return Storage.schema().parse(value); }
}
    `},
	}, []rule_tester.InvalidTestCase{})
}
