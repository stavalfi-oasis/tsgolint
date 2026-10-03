package prefer_super_over_this

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestPreferSuperOverThis(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &PreferSuperOverThisRule, []rule_tester.ValidTestCase{
		// Declared on the class itself.
		{Code: "class A { run(): void {} } class B extends A { run(): void {} go(): void { this.run(); } }"},
		// No base class at all.
		{Code: "class A { run(): void {} go(): void { this.run(); } }"},
		// An instance field is not on the prototype — `super.run` is undefined.
		{Code: "class A { run = (): void => {}; } class B extends A { go(): void { this.run(); } }"},
		// An abstract method has no base implementation to dispatch to.
		{Code: "abstract class A { abstract run(): void; } class B extends A { run(): void {} go(): void { this.run(); } }"},
		// Static members.
		{Code: "class A { static run(): void {} } class B extends A { static go(): void { this.run(); } }"},
		// Passed around as a value rather than called.
		{Code: "class A { run(): void {} } class B extends A { go(): () => void { return this.run.bind(this); } }"},
		// `function` rebinds `this` and makes `super` a syntax error.
		{Code: "class A { run(): void {} } class B extends A { go(): void { const f = function (this: B) { this.run(); }; f.call(this); } }"},
		// An interface member is not a prototype member of a base class.
		{Code: "interface I { run(): void } class A {} class B extends A implements I { run(): void {} go(): void { this.run(); } }"},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   "class A { run(): void {} } class B extends A { go(): void { this.run(); } }",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferSuper"}},
			Output: []string{"class A { run(): void {} } class B extends A { go(): void { super.run(); } }"},
		},
		{
			// An arrow function keeps both `this` and `super`.
			Code:   "class A { run(): void {} } class B extends A { go(): void { const f = (): void => { this.run(); }; f(); } }",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferSuper"}},
			Output: []string{"class A { run(): void {} } class B extends A { go(): void { const f = (): void => { super.run(); }; f(); } }"},
		},
		{
			// A getter inherited from the base class.
			Code:   "class A { get name(): string { return 'a'; } } class B extends A { go(): string { return this.name; } }",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferSuper"}},
			Output: []string{"class A { get name(): string { return 'a'; } } class B extends A { go(): string { return super.name; } }"},
		},
		{
			// Two levels up the chain.
			Code:   "class A { run(): void {} } class B extends A {} class C extends B { go(): void { this.run(); } }",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferSuper"}},
			Output: []string{"class A { run(): void {} } class B extends A {} class C extends B { go(): void { super.run(); } }"},
		},
		{
			// Inside a constructor.
			Code:   "class A { run(): void {} } class B extends A { constructor() { super(); this.run(); } }",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferSuper"}},
			Output: []string{"class A { run(): void {} } class B extends A { constructor() { super(); super.run(); } }"},
		},
	})
}
