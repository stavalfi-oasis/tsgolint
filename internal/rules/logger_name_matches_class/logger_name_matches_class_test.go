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

// A second stub, so the cases added for the missing-child half of the rule do
// not edit the text of the cases above: snapshots are keyed by case index and
// rendered from the code, so changing `stub` would rewrite every one of them.
const loggerStub = `
interface Logger { child(bindings: { name?: string; service?: string }): Logger }
interface Options { readonly logger: Logger }
`

func TestLoggerNameMatchesClass(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &LoggerNameMatchesClassRule, []rule_tester.ValidTestCase{
		{Code: stub + `class GatewayService { public constructor() { logger.logger.child({ name: GatewayService.name }); } }`},
		{Code: stub + `class GatewayService { public run(): void { logger.logger.child({ name: "anything" }); } }`},
		{Code: loggerStub + `class GatewayService { #logger: Logger; public constructor(logger: Logger) { this.#logger = logger.child({ name: GatewayService.name }); } }`},
		{Code: loggerStub + `class GatewayService { #logger: Logger; public constructor(options: Options) { this.#logger = options.logger.child({ name: GatewayService.name, service: "gh-gateway" }); } }`},
		{Code: loggerStub + `class GatewayService { #logger: Logger; public constructor({ logger }: Options) { this.#logger = logger.child({ name: GatewayService.name }); } }`},
		{Code: loggerStub + `class GatewayService { public constructor(port: number) { void port; } }`},
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
		{
			Code:   loggerStub + `class GatewayService { #logger: Logger; public constructor(logger: Logger) { this.#logger = logger; } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "loggerChildMissing"}},
		},
		{
			Code:   loggerStub + `class GatewayService { #logger: Logger; public constructor(options: Options) { this.#logger = options.logger; } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "loggerChildMissing"}},
		},
		{
			Code:   loggerStub + `class GatewayService { #logger: Logger; public constructor({ logger }: Options) { this.#logger = logger.child({ service: "gh-gateway" }); } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "loggerChildMissing"}},
		},
	})
}
