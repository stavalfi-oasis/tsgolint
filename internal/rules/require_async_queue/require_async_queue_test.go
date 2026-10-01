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
		// The shape the rule is asking for.
		{Code: stub + `
class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super();
    this.#asyncQueue = asyncQueue;
  }
  run(): AsyncQueue { return this.#asyncQueue; }
}
    `},
		// Alongside other options.
		{Code: stub + `
class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;
  constructor({
    asyncQueue,
    logger,
  }: {
    readonly asyncQueue: AsyncQueue;
    readonly logger: Logger;
  }) {
    super();
    this.#asyncQueue = asyncQueue;
  }
  run(): AsyncQueue { return this.#asyncQueue; }
}
    `},
		// Assigned from a renamed binding.
		{Code: stub + `
class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;
  constructor({ asyncQueue: queue }: { readonly asyncQueue: AsyncQueue }) {
    super();
    this.#asyncQueue = queue;
  }
  run(): AsyncQueue { return this.#asyncQueue; }
}
    `},
		// Reached through a whole options object rather than destructured.
		{Code: stub + `
class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;
  constructor(options: { readonly asyncQueue: AsyncQueue }) {
    super();
    this.#asyncQueue = options.asyncQueue;
  }
  run(): AsyncQueue { return this.#asyncQueue; }
}
    `},
		// A `this` parameter is a type annotation, not the options object.
		{Code: stub + `
class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;
  constructor(this: Service, { asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super();
    this.#asyncQueue = asyncQueue;
  }
  run(): AsyncQueue { return this.#asyncQueue; }
}
    `},
		// Extends a subclass of ADisposable, which already satisfies the rule.
		{Code: stub + `
class Middle extends ADisposable {
  readonly #asyncQueue: AsyncQueue;
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super();
    this.#asyncQueue = asyncQueue;
  }
  run(): AsyncQueue { return this.#asyncQueue; }
}
class Service extends Middle {
  readonly #asyncQueue: AsyncQueue;
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super({ asyncQueue });
    this.#asyncQueue = asyncQueue;
  }
  go(): AsyncQueue { return this.#asyncQueue; }
}
    `},
		// A class expression, which the JS rules never visited.
		{Code: stub + `
const Service = class extends ADisposable {
  readonly #asyncQueue: AsyncQueue;
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super();
    this.#asyncQueue = asyncQueue;
  }
  run(): AsyncQueue { return this.#asyncQueue; }
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
  readonly #asyncQueue: AsyncQueue;
  constructor({ asyncQueue }: ToolArgs) {
    super();
    this.#asyncQueue = asyncQueue;
  }
  run(): AsyncQueue { return this.#asyncQueue; }
}
    `},
		// Same, through an intersection: the property sits on one side of it.
		{Code: stub + `
interface Tuning {
  readonly asyncQueue: AsyncQueue;
}
class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;
  constructor(options: Tuning & { readonly logger: Logger }) {
    super();
    this.#asyncQueue = options.asyncQueue;
  }
  run(): AsyncQueue { return this.#asyncQueue; }
}
    `},
		// A class that does not extend ADisposable is none of this rule's business.
		{Code: stub + `
class Plain {
  constructor({ logger }: { readonly logger: Logger }) {}
}
    `},
		// ADisposable itself is exempt: it has no super to accept a queue from.
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
  readonly #asyncQueue: AsyncQueue;

  constructor({ asyncQueue,
  logger }: { readonly asyncQueue: AsyncQueue;
  readonly logger: Logger }) {
    super();
    this.#asyncQueue = asyncQueue;
  }

  public get asyncQueue(): AsyncQueue {
    return this.#asyncQueue;
  }
}
      `},
		},
		// Accepted and dropped on the floor: only the storing half is missing.
		{
			Code: stub + `
class Service extends ADisposable {
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingField"}},
			Output: []string{`
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class AsyncQueue {}
declare class Logger {}

class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;

  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super();
    this.#asyncQueue = asyncQueue;
  }

  public get asyncQueue(): AsyncQueue {
    return this.#asyncQueue;
  }
}
      `},
		},
		// The field is already declared and read: the fix adds the assignment it
		// is missing and nothing else — no second field, no second getter.
		{
			Code: stub + `
class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super();
  }
  run(): AsyncQueue { return this.#asyncQueue; }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingField"}},
			Output: []string{`
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class AsyncQueue {}
declare class Logger {}

class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super();
    this.#asyncQueue = asyncQueue;
  }
  run(): AsyncQueue { return this.#asyncQueue; }
}
      `},
		},
		// From the other direction: field and assignment are already there, only
		// the option is missing. Neither may be declared a second time.
		{
			Code: stub + `
class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;
  constructor({ logger }: { readonly logger: Logger }) {
    super();
    this.#asyncQueue = asyncQueue;
  }
  run(): AsyncQueue { return this.#asyncQueue; }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingOption"}},
			Output: []string{`
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class AsyncQueue {}
declare class Logger {}

class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;
  constructor({ asyncQueue,
  logger }: { readonly asyncQueue: AsyncQueue;
  readonly logger: Logger }) {
    super();
    this.#asyncQueue = asyncQueue;
  }
  run(): AsyncQueue { return this.#asyncQueue; }
}
      `},
		},
		// A whole options object rather than a destructuring pattern: the
		// assignment has to reach through the parameter, not name a binding
		// that does not exist.
		{
			Code: stub + `
class Service extends ADisposable {
  constructor(options: { readonly asyncQueue: AsyncQueue }) {
    super();
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingField"}},
			Output: []string{`
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class AsyncQueue {}
declare class Logger {}

class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;

  constructor(options: { readonly asyncQueue: AsyncQueue }) {
    super();
    this.#asyncQueue = options.asyncQueue;
  }

  public get asyncQueue(): AsyncQueue {
    return this.#asyncQueue;
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
    super();
  }
}
      `,
				`
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class AsyncQueue {}
declare class Logger {}

class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;

  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super();
    this.#asyncQueue = asyncQueue;
  }

  public get asyncQueue(): AsyncQueue {
    return this.#asyncQueue;
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
  readonly #asyncQueue: AsyncQueue;

  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue; }) {
    super();
    this.#asyncQueue = asyncQueue;
  }

  public get asyncQueue(): AsyncQueue {
    return this.#asyncQueue;
  }
}
      `},
		},
		// An empty constructor body, so the assignment has nothing to follow.
		{
			Code: stub + `
class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {}
  run(): AsyncQueue { return this.#asyncQueue; }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingField"}},
			Output: []string{`
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class AsyncQueue {}
declare class Logger {}

class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super();
    this.#asyncQueue = asyncQueue;
  }
  run(): AsyncQueue { return this.#asyncQueue; }
}
      `},
		},
		// No constructor at all, with members to insert ahead of.
		{
			Code: stub + `
class Service extends ADisposable {
  run(): void {}
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingOption"}},
			Output: []string{`
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class AsyncQueue {}
declare class Logger {}

class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;

  public constructor({
    asyncQueue,
  }: {
    readonly asyncQueue: AsyncQueue;
  }) {
    super();
    this.#asyncQueue = asyncQueue;
  }

  public get asyncQueue(): AsyncQueue {
    return this.#asyncQueue;
  }

  run(): void {}
}
      `},
		},
		// No constructor and no members at all: the whole body is written.
		{
			Code: stub + `
class Service extends ADisposable {}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingOption"}},
			Output: []string{`
import { ADisposable } from "./a-disposable.ts";
import type { AsyncQueue } from "./async-queue.ts";
declare class AsyncQueue {}
declare class Logger {}

class Service extends ADisposable {
  readonly #asyncQueue: AsyncQueue;

  public constructor({
    asyncQueue,
  }: {
    readonly asyncQueue: AsyncQueue;
  }) {
    super();
    this.#asyncQueue = asyncQueue;
  }

  public get asyncQueue(): AsyncQueue {
    return this.#asyncQueue;
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
  readonly #asyncQueue: AsyncQueue;

  constructor({ asyncQueue,
  logger }: { readonly asyncQueue: AsyncQueue;
  readonly logger: Logger }) {
    super();
    this.#asyncQueue = asyncQueue;
  }

  public get asyncQueue(): AsyncQueue {
    return this.#asyncQueue;
  }
};
      `},
		},
		// Extends a subclass of ADisposable: the transitive case the name-only
		// check the JS rules used would have missed. ADisposable is declared
		// locally here on purpose — reached through an unresolvable import the
		// type walk cannot see the base chain, and only the direct subclass
		// would be reported.
		{
			Code: `
declare class AsyncQueue {}
declare class Logger {}
class ADisposable {}
class Middle extends ADisposable {
  readonly #asyncQueue: AsyncQueue;
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super();
    this.#asyncQueue = asyncQueue;
  }
  run(): AsyncQueue { return this.#asyncQueue; }
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
  readonly #asyncQueue: AsyncQueue;
  constructor({ asyncQueue }: { readonly asyncQueue: AsyncQueue }) {
    super();
    this.#asyncQueue = asyncQueue;
  }
  run(): AsyncQueue { return this.#asyncQueue; }
}
class Service extends Middle {
  readonly #asyncQueue: AsyncQueue;

  constructor({ asyncQueue,
  logger }: { readonly asyncQueue: AsyncQueue;
  readonly logger: Logger }) {
    super({ asyncQueue: null as unknown as AsyncQueue });
    this.#asyncQueue = asyncQueue;
  }

  public get asyncQueue(): AsyncQueue {
    return this.#asyncQueue;
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
  readonly #asyncQueue: AsyncQueue;

  constructor({ asyncQueue,
  logger }: { readonly asyncQueue: AsyncQueue;
  readonly logger: Logger }) {
    super();
    this.#asyncQueue = asyncQueue;
  }

  public get asyncQueue(): AsyncQueue {
    return this.#asyncQueue;
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
  readonly #asyncQueue: AsyncQueue;

  constructor({ asyncQueue,
  logger }: { readonly asyncQueue: AsyncQueue;
  readonly logger: Logger }) {
    super();
    this.#asyncQueue = asyncQueue;
  }

  public get asyncQueue(): AsyncQueue {
    return this.#asyncQueue;
  }
}
      `},
		},
	})
}
