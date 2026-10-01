package require_async_queue

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

const stub = `
import { ADisposable } from "./a-disposable.ts";
declare class AsyncQueue {}
declare class Logger {}
`

const fixedStub = `
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class AsyncQueue {}
declare class Logger {}
`

func TestRequireAsyncQueue(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireAsyncQueueRule, []rule_tester.ValidTestCase{
		{Code: stub + `
class Service extends ADisposable {
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super({ asyncQueue });
  }
}
    `},
		{Code: stub + `
class Service extends ADisposable {
  constructor({
    asyncQueue,
    logger,
  }: {
    readonly asyncQueue: AsyncQueue;
    readonly logger: Logger;
  }) {
    super({ asyncQueue });
  }
}
    `},
		// A class that does not extend ADisposable is none of this rule's business.
		{Code: stub + `
class Plain {
  constructor({ logger }: { readonly logger: Logger }) {}
}
    `},
	}, []rule_tester.InvalidTestCase{
		// Options object that is missing the queue: the property, the binding and
		// the super() call all have to move together, or the fix leaves code that
		// does not compile.
		{
			Code: stub + `
class Service extends ADisposable {
  constructor({ logger }: { readonly logger: Logger }) {
    super();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingOption"},
			},
			Output: []string{fixedStub + `
class Service extends ADisposable {
  constructor({ asyncQueue,
  logger }: { readonly asyncQueue: AsyncQueue;
  readonly logger: Logger }) {
    super({ asyncQueue });
  }
}
      `},
		},
		// A constructor with no parameters at all gets the whole options object.
		{
			Code: stub + `
class Service extends ADisposable {
  constructor() {
    super();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingOption"},
			},
			Output: []string{fixedStub + `
class Service extends ADisposable {
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super({ asyncQueue });
  }
}
      `},
		},
		// No constructor at all: one is written, ahead of the existing members.
		{
			Code: stub + `
class Service extends ADisposable {
  run(): void {}
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingOption"},
			},
			Output: []string{fixedStub + `
class Service extends ADisposable {
  public constructor({
    asyncQueue,
  }: {
    readonly asyncQueue: AsyncQueue;
  }) {
    super({ asyncQueue });
  }

  run(): void {}
}
      `},
		},
	})
}
