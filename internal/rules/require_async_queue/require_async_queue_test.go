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

func TestRequireAsyncQueue(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireAsyncQueueRule, []rule_tester.ValidTestCase{
		// The shape the rule is asking for: accepted and handed to the base,
		// which is the one place the queue is now kept.
		{Code: stub + `
class Service extends ADisposable {
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super({ asyncQueue });
  }
}
    `},
		// Alongside other options.
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
		// Renamed binding: the option is what the rule asks for, not the name it
		// is destructured under.
		{Code: stub + `
class Service extends ADisposable {
  constructor({ asyncQueue: queue }: { readonly asyncQueue: AsyncQueue }) {
    super({ asyncQueue: queue });
  }
}
    `},
		// Reached through a whole options object rather than destructured.
		{Code: stub + `
class Service extends ADisposable {
  constructor(options: { readonly asyncQueue: AsyncQueue }) {
    super(options);
  }
}
    `},
		// A `this` parameter is a type annotation, not the options object.
		{Code: stub + `
class Service extends ADisposable {
  constructor(this: Service, { asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super({ asyncQueue });
  }
}
    `},
		// Extends a subclass of ADisposable, which already satisfies the rule.
		{Code: stub + `
class Middle extends ADisposable {
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super({ asyncQueue });
  }
}
class Service extends Middle {
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super({ asyncQueue });
  }
}
    `},
		// A class expression, which the JS rules never visited.
		{Code: stub + `
const Service = class extends ADisposable {
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super({ asyncQueue });
  }
};
    `},
		// The option arrives through a named interface. The annotation does not
		// spell it, so only the checker can see it.
		{Code: stub + `
interface ToolArgs {
  readonly asyncQueue: AsyncQueue;
  readonly logger: Logger;
}
class Service extends ADisposable {
  constructor({ asyncQueue }: ToolArgs) {
    super({ asyncQueue });
  }
}
    `},
		// Same, through an intersection: the property sits on one side of it.
		{Code: stub + `
interface Tuning {
  readonly asyncQueue: AsyncQueue;
}
class Service extends ADisposable {
  constructor(options: Tuning & { readonly logger: Logger }) {
    super(options);
  }
}
    `},
		// An optional options parameter is a union with undefined, so the type
		// carries no properties and only the annotation answers.
		{Code: stub + `
class Service extends ADisposable {
  constructor(args?: { readonly asyncQueue: AsyncQueue; readonly logger?: Logger }) {
    super({ asyncQueue: args?.asyncQueue as unknown as AsyncQueue });
  }
}
    `},
		// No constructor at all: the implicit one forwards to ADisposable's,
		// which already demands the queue, so there is nothing to add.
		{Code: stub + `
class Service extends ADisposable {
  run(): void {}
}
    `},
		// A class that does not extend ADisposable is none of this rule's business.
		{Code: stub + `
class Plain {
  constructor({ logger }: { readonly logger: Logger }) {}
}
    `},
		// ADisposable itself is exempt: it is the class that holds the queue.
		{Code: `
class ADisposable {
  run(): void {}
}
    `},
	}, []rule_tester.InvalidTestCase{
		// Options object missing the queue entirely.
		{
			Code: stub + `
class Service extends ADisposable {
  constructor({ logger }: { readonly logger: Logger }) {
    super();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingOption"}},
			Output: []string{`
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class AsyncQueue {}
declare class Logger {}

class Service extends ADisposable {
  constructor({ asyncQueue,
  logger }: { readonly asyncQueue: AsyncQueue;
  readonly logger: Logger }) {
    super({ asyncQueue });
  }
}
      `},
		},
		// A whole options object rather than a destructuring pattern: the
		// super call takes the object it already has.
		{
			Code: stub + `
class Service extends ADisposable {
  constructor(options: { readonly logger: Logger }) {
    super();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingOption"}},
			Output: []string{`
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class AsyncQueue {}
declare class Logger {}

class Service extends ADisposable {
  constructor(options: { readonly asyncQueue: AsyncQueue;
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
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingOption"}},
			Output: []string{`
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class AsyncQueue {}
declare class Logger {}

class Service extends ADisposable {
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super({ asyncQueue });
  }
}
      `},
		},
		// An empty options type literal: the property goes in between the braces.
		{
			Code: stub + `
class Service extends ADisposable {
  constructor({}: {}) {
    super();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingOption"}},
			Output: []string{`
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class AsyncQueue {}
declare class Logger {}

class Service extends ADisposable {
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue; }) {
    super({ asyncQueue });
  }
}
      `},
		},
		// An empty constructor body, so the super call has nothing to follow.
		{
			Code: stub + `
class Service extends ADisposable {
  constructor({ logger }: { readonly logger: Logger }) {}
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingOption"}},
			Output: []string{`
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class AsyncQueue {}
declare class Logger {}

class Service extends ADisposable {
  constructor({ asyncQueue,
  logger }: { readonly asyncQueue: AsyncQueue;
  readonly logger: Logger }) {
    super({ asyncQueue });
  }
}
      `},
		},
		// A named options interface is shared with other declarations, so the
		// rule reports it and leaves the edit to a human.
		{
			Code: stub + `
interface ServiceOptions {
  readonly logger: Logger;
}
class Service extends ADisposable {
  constructor(options: ServiceOptions) {
    super();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingOption"}},
		},
		// A class expression is reported like a declaration.
		{
			Code: stub + `
const Service = class extends ADisposable {
  constructor({ logger }: { readonly logger: Logger }) {
    super();
  }
};
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingOption"}},
			Output: []string{`
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class AsyncQueue {}
declare class Logger {}

const Service = class extends ADisposable {
  constructor({ asyncQueue,
  logger }: { readonly asyncQueue: AsyncQueue;
  readonly logger: Logger }) {
    super({ asyncQueue });
  }
};
      `},
		},
		// Extends a subclass of ADisposable: the transitive case the name-only
		// check the JS rules used would have missed. The super call already has
		// an argument, so only the option is added.
		{
			Code: `
declare class AsyncQueue {}
declare class Logger {}
class ADisposable {}
class Middle extends ADisposable {
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super();
  }
}
class Service extends Middle {
  constructor({ logger }: { readonly logger: Logger }) {
    super({ asyncQueue: null as unknown as AsyncQueue });
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingOption"}},
			Output: []string{`
declare class AsyncQueue {}
declare class Logger {}
class ADisposable {}
class Middle extends ADisposable {
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super();
  }
}
class Service extends Middle {
  constructor({ asyncQueue,
  logger }: { readonly asyncQueue: AsyncQueue;
  readonly logger: Logger }) {
    super({ asyncQueue: null as unknown as AsyncQueue });
  }
}
      `},
		},
		// AsyncQueue is already imported, so the fix must not import it twice.
		{
			Code: `
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class Logger {}
class Service extends ADisposable {
  constructor({ logger }: { readonly logger: Logger }) {
    super();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingOption"}},
			Output: []string{`
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class Logger {}
class Service extends ADisposable {
  constructor({ asyncQueue,
  logger }: { readonly asyncQueue: AsyncQueue;
  readonly logger: Logger }) {
    super({ asyncQueue });
  }
}
      `},
		},
		// No ADisposable import to model the specifier on — the code fix still
		// runs, it just cannot add an import it would have to invent a path for.
		{
			Code: `
declare class ADisposable {}
declare class AsyncQueue {}
declare class Logger {}
class Service extends ADisposable {
  constructor({ logger }: { readonly logger: Logger }) {
    super();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingOption"}},
			Output: []string{`
declare class ADisposable {}
declare class AsyncQueue {}
declare class Logger {}
class Service extends ADisposable {
  constructor({ asyncQueue,
  logger }: { readonly asyncQueue: AsyncQueue;
  readonly logger: Logger }) {
    super({ asyncQueue });
  }
}
      `},
		},
	})
}
