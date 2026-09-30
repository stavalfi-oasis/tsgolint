package no_unused_public_member

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoUnusedPublicMember(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoUnusedPublicMemberRule, []rule_tester.ValidTestCase{
		{Code: `
class Service {
  public run(): void {}
}
const service = new Service();
service.run();
    `},
		{Code: `
class Service {
  public static make(): Service { return new Service(); }
}
const service = Service.make();
    `},
		// Already private: the rule has nothing to say.
		{Code: `
class Service {
  #run(): void {}
  public start(): void { this.#run(); }
}
const service = new Service();
service.start();
    `},
		// The member is the contract's shape, not a free choice.
		{Code: `
interface Runnable { run(): void }
class Service implements Runnable {
  public run(): void {}
}
    `},
		// Taken off the class by a destructuring binding, which is how the
		// Temporal workflow entrypoints are exported.
		{Code: `
class Service {
  public static run(): void {}
}
const { run } = Service;
void run;
    `},
		// A computed name is never referenced by that name.
		{Code: `
class Service {
  public async [Symbol.asyncDispose](): Promise<void> {}
}
    `},
		// An exported class can be used from a file this program does not
		// contain, so absence of a use proves nothing.
		{Code: `
export class Service {
  public helper(): number { return 1; }
  public run(): number { return this.helper(); }
}
    `},
	}, []rule_tester.InvalidTestCase{
		{
			// Called only through `this`, from inside its own class.
			Code: `
class Service {
  public helper(): number { return 1; }
  public run(): number { return this.helper(); }
}
const service = new Service();
service.run();
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "unusedPublicMember"}},
		},
		{
			// Nothing calls it at all.
			Code: `
class Service {
  public dead(): void {}
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "unusedPublicMember"}},
		},
		{
			Code: `
class Service {
  public static helper(): number { return 1; }
  public run(): number { return Service.helper(); }
}
const service = new Service();
service.run();
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "unusedPublicMember"}},
		},
	})
}
