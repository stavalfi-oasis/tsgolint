package no_protected

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

// The exempt case compiles under its own file name. The fixtures tsconfig globs
// the directory and the base VFS is cached across cases, so every other case
// declares that name too — otherwise a leftover directory listing names a file
// the overlay does not have.
var exemptFile = map[string]string{disposableBaseFile: "export {};"}

func TestNoProtected(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoProtectedRule, []rule_tester.ValidTestCase{
		{Code: `class A { #value = 1; }`, Files: exemptFile},
		{
			FileName: disposableBaseFile,
			Code:     `class ADisposable { protected async close(): Promise<void> {} }`,
		},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `class A { protected value = 1; }`,
			Files:  exemptFile,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "protectedMember"}},
		},
		{
			Code:   `class A { protected read(): void {} }`,
			Files:  exemptFile,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "protectedMember"}},
		},
	})
}
