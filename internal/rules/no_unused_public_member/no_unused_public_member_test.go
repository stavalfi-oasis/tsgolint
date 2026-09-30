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
export class Service {
  public run(): void {}
}
const service = new Service();
service.run();
    `},
		{Code: `
export class Service {
  public static make(): Service { return new Service(); }
}
const service = Service.make();
    `},
		// Already private: the rule has nothing to say.
		{Code: `
export class Service {
  #run(): void {}
  public start(): void { this.#run(); }
}
const service = new Service();
service.start();
    `},
		// The member is the contract's shape, not a free choice.
		{Code: `
interface Runnable { run(): void }
export class Service implements Runnable {
  public run(): void {}
}
    `},
		// A computed name is never referenced by that name.
		{Code: `
export class Service {
  public async [Symbol.asyncDispose](): Promise<void> {}
}
    `},
	}, []rule_tester.InvalidTestCase{
		{
			// Called only through `this`, from inside its own class.
			Code: `
export class Service {
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
export class Service {
  public dead(): void {}
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "unusedPublicMember"}},
		},
		{
			Code: `
export class Service {
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
