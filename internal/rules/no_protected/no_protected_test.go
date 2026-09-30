package no_protected

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoProtected(t *testing.T) {
	// Not parallel: the exempt-file case compiles under its own file name, and
	// the fixtures tsconfig globs the whole directory.
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoProtectedRule, []rule_tester.ValidTestCase{
		{Code: `class A { #value = 1; }`},
		{
			FileName: "shared/libs/src/a-disposable.ts",
			Code:     `class ADisposable { protected async close(): Promise<void> {} }`,
		},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `class A { protected value = 1; }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "protectedMember"}},
		},
		{
			Code:   `class A { protected read(): void {} }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "protectedMember"}},
		},
	})
}
