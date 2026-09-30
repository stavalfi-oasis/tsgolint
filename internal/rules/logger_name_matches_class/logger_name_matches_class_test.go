package logger_name_matches_class

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

const stub = `
interface Child { child(bindings: { name: string }): Child }
declare const logger: { logger: Child };
`

func TestLoggerNameMatchesClass(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &LoggerNameMatchesClassRule, []rule_tester.ValidTestCase{
		{Code: stub + `class GatewayService { public constructor() { logger.logger.child({ name: GatewayService.name }); } }`},
		{Code: stub + `class GatewayService { public run(): void { logger.logger.child({ name: "anything" }); } }`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   stub + `class GatewayService { public constructor() { logger.logger.child({ name: "gateway" }); } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "loggerNameMismatch"}},
			Output: []string{stub + `class GatewayService { public constructor() { logger.logger.child({ name: GatewayService.name }); } }`},
		},
		{
			Code:   stub + `class GatewayService { public constructor() { logger.logger.child({ name: OtherService.name }); } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "loggerNameMismatch"}},
			Output: []string{stub + `class GatewayService { public constructor() { logger.logger.child({ name: GatewayService.name }); } }`},
		},
	})
}
