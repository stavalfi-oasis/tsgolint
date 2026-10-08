package new_only_in_constructor

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNewOnlyInConstructor(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NewOnlyInConstructorRule, []rule_tester.ValidTestCase{
		{Code: `class Dep {} class A { #dep: Dep; constructor() { this.#dep = new Dep(); } }`},
		{Code: `class Dep {} class A { #dep = new Dep(); }`},
		// a property initializer runs as part of construction, so it is the constructor.
		{Code: `class Dep {} class A { #dep: Dep = new Dep(); }`},
		// outside any class this rule has no opinion.
		{Code: `class Dep {} const dep = new Dep();`},
		{Code: `class Dep {} function make(): Dep { return new Dep(); }`},
		{Code: `class Dep {} const make = (): Dep => new Dep();`},
		// `allow` names the classes that may be constructed anywhere.
		{
			Code:    `class Dep {} class A { start(): Dep { return new Dep(); } }`,
			Options: NewOnlyInConstructorOptions{Allow: []string{"Dep"}},
		},
		{
			Code:    `class A { fail(): never { throw new Error("x"); } }`,
			Options: NewOnlyInConstructorOptions{Allow: []string{"Error"}},
		},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `class Dep {} class A { #dep?: Dep; start(): void { this.#dep = new Dep(); } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "newOutsideConstructor"}},
		},
		{
			Code:   `class Dep {} class A { get dep(): Dep { return new Dep(); } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "newOutsideConstructor"}},
		},
		{
			Code:   `class Dep {} class A { set dep(value: Dep) { value = new Dep(); } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "newOutsideConstructor"}},
		},
		{
			Code:   `class Dep {} class A { static make(): Dep { return new Dep(); } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "newOutsideConstructor"}},
		},
		// nested in a callback inside a method: still outside the constructor.
		{
			Code:   `class Dep {} class A { start(): () => Dep { return () => new Dep(); } }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "newOutsideConstructor"}},
		},
		{
			Code:   `class Dep {} const A = class { start(): Dep { return new Dep(); } };`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "newOutsideConstructor"}},
		},
		// `allow` lists a different class, so this one is still reported.
		{
			Code:    `class Dep {} class Other {} class A { start(): Dep { return new Dep(); } }`,
			Options: NewOnlyInConstructorOptions{Allow: []string{"Other"}},
			Errors:  []rule_tester.InvalidTestCaseError{{MessageId: "newOutsideConstructor"}},
		},
		// a namespaced constructor is named by its last segment.
		{
			Code:    `namespace ns { export class Dep {} } class A { start(): ns.Dep { return new ns.Dep(); } }`,
			Options: NewOnlyInConstructorOptions{Allow: []string{"ns"}},
			Errors:  []rule_tester.InvalidTestCaseError{{MessageId: "newOutsideConstructor"}},
		},
	})
}
