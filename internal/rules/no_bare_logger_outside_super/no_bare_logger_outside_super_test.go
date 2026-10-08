package no_bare_logger_outside_super

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
  protected readonly logger: Logger;
  close(): Promise<void>;
}
declare class Inner {
  constructor(options: { readonly logger: Logger });
}
`

func TestNoBareLoggerOutsideSuper(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoBareLoggerOutsideSuperRule, []rule_tester.ValidTestCase{
		{Code: disposableStub + `
class Service extends ADisposable {
  readonly #inner: Inner;

  constructor({ logger }: { readonly logger: Logger }) {
    super({ logger: logger.child({ name: Service.name }) });
    this.#inner = new Inner({ logger: this.logger });
  }
}
    `},
		// The key `logger:` is a name, not a reference.
		{Code: disposableStub + `
class Service extends ADisposable {
  constructor({ logger }: { readonly logger: Logger }) {
    super({ logger: logger.child({ name: Service.name }) });
  }
}
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: disposableStub + `
class Service extends ADisposable {
  readonly #inner: Inner;

  constructor({ logger }: { readonly logger: Logger }) {
    super({ logger: logger.child({ name: Service.name }) });
    this.#inner = new Inner({ logger });
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "bareLogger"}},
		},
		{
			Code: disposableStub + `
class Service extends ADisposable {
  readonly #inner: Inner;

  constructor({ logger }: { readonly logger: Logger }) {
    super({ logger: logger.child({ name: Service.name }) });
    this.#inner = new Inner({ logger: logger });
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "bareLogger"}},
		},
	})
}
