package require_logger_child_in_super

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

const disposableStub = `
declare class Logger {
  child(bindings: { readonly name?: string; readonly service?: string }): Logger;
}
declare class ADisposable {
  constructor(options: { readonly logger: Logger });
  close(): Promise<void>;
}
`

func TestRequireLoggerChildInSuper(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireLoggerChildInSuperRule, []rule_tester.ValidTestCase{
		{Code: disposableStub + `
class Service extends ADisposable {
  constructor({ logger }: { readonly logger: Logger }) {
    super({ logger: logger.child({ name: Service.name }) });
  }
}
    `},
		// Extra bindings alongside name are fine.
		{Code: disposableStub + `
class Service extends ADisposable {
  constructor({ logger }: { readonly logger: Logger }) {
    super({ logger: logger.child({ name: Service.name, service: "gh-gateway" }) });
  }
}
    `},
		// No constructor of its own — it inherits the base's.
		{Code: disposableStub + `
class Service extends ADisposable {
  run(): void {}
}
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: disposableStub + `
class Service extends ADisposable {
  constructor({ logger }: { readonly logger: Logger }) {
    super({ logger });
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingLoggerChild"}},
		},
		// child() without a name binding labels every line after the base class.
		{
			Code: disposableStub + `
class Service extends ADisposable {
  constructor({ logger }: { readonly logger: Logger }) {
    super({ logger: logger.child({ service: "gh-gateway" }) });
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingLoggerChild"}},
		},
	})
}
