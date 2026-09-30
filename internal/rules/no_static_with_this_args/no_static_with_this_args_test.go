package no_static_with_this_args

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoStaticWithThisArgs(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoStaticWithThisArgsRule, []rule_tester.ValidTestCase{
		// No instance state handed over.
		{Code: `
class Service {
  static build(port: number): string { return String(port); }
  run(): string { return Service.build(8080); }
}
    `},
		// One call passes instance state, another does not — genuinely static.
		{Code: `
class Service {
  #port = 1;
  static build(port: number): string { return String(port); }
  a(): string { return Service.build(this.#port); }
  b(): string { return Service.build(8080); }
}
    `},
		// Not static.
		{Code: `
class Service {
  #port = 1;
  build(port: number): string { return String(port); }
  run(): string { return this.build(this.#port); }
}
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: `
class Service {
  #port = 1;
  static build(port: number): string { return String(port); }
  run(): string { return Service.build(this.#port); }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "staticWithThisArgs", Line: 5, Column: 26, EndLine: 5, EndColumn: 51},
			},
		},
		// Reached through an alias. The JS rule matched the literal text
		// `ClassName.member`, so this call site was invisible to it.
		{
			Code: `
class Service {
  #port = 1;
  static build(port: number): string { return String(port); }
  run(): string {
    const Alias = Service;
    return Alias.build(this.#port);
  }
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "staticWithThisArgs", Line: 7, Column: 12, EndLine: 7, EndColumn: 35},
			},
		},
	})
}

func TestNoStaticWithThisArgsForeignClass(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoStaticWithThisArgsRule, []rule_tester.ValidTestCase{
		// A static on another class names a factory the call site cannot edit:
		// there is no instance to move it onto.
		{Code: `
declare class Worker {
  static create(options: { namespace: string }): Worker;
}
class Runner {
  #namespace = "default";
  run(): Worker { return Worker.create({ namespace: this.#namespace }); }
}
    `},
	}, []rule_tester.InvalidTestCase{})
}
