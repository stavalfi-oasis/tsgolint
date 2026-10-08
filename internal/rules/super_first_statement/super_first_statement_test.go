package super_first_statement

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

const base = `
declare class Base {
  constructor(options: { readonly value: number });
}
declare function work(): number;
`

func TestSuperFirstStatement(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &SuperFirstStatementRule, []rule_tester.ValidTestCase{
		{Code: base + `
class Service extends Base {
  constructor() {
    super({ value: 1 });
  }
}
    `},
		{Code: base + `
class Service extends Base {
  constructor() {
    super({ value: work() });
    work();
  }
}
    `},
		// No super at all — a base class of its own.
		{Code: base + `
class Service {
  constructor() {
    work();
  }
}
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: base + `
class Service extends Base {
  constructor() {
    const value = work();
    super({ value });
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "superNotFirst", Line: 10, Column: 5, EndLine: 10, EndColumn: 22},
			},
		},
	})
}
