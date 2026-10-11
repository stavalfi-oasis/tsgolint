package require_constant_env_key

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestRequireConstantEnvKey(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireConstantEnvKeyRule, []rule_tester.ValidTestCase{
		{Code: `const port = process.env["PORT"];`},
		{Code: `const port = process.env.PORT;`},
		{Code: `declare const env: Record<string, string>; const port = env["PORT"];`},
		{Code: `declare const config: Record<string, string>; declare const key: string; const value = config[key];`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   "declare const name: string; const port = process.env[name];",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "constantEnvKey"}},
		},
		{
			Code:   "declare const prefix: string; const port = process.env[`${prefix}_PORT`];",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "constantEnvKey"}},
		},
		{
			Code:   "const port = process.env[`PORT`];",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "constantEnvKey"}},
		},
		{
			Code:   `declare const env: Record<string, string>; declare const name: string; const port = env[name];`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "constantEnvKey"}},
		},
	})
}
