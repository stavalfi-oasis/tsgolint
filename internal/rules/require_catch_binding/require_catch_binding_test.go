package require_catch_binding

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestRequireCatchBinding(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireCatchBindingRule, []rule_tester.ValidTestCase{
		{Code: `try {} catch (error) {}`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `try {} catch {}`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingCatchBinding"}},
			Output: []string{`try {} catch (error) {}`},
		},
	})
}
