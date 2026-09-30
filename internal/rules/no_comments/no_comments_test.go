package no_comments

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoComments(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoCommentsRule, []rule_tester.ValidTestCase{
		{Code: "const value = 1;\n"},
		// Directives the tools need in order to work stay.
		{Code: "// oxlint-disable-next-line no-console\nconst value = 1;\n"},
		{Code: "// @ts-expect-error\nconst value = 1;\n"},
		{Code: "/// <reference types=\"node\" />\nconst value = 1;\n"},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   "// explains the next line\nconst value = 1;\n",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "comment"}},
			Output: []string{"const value = 1;\n"},
		},
		{
			Code:   "const value = 1; // trailing\n",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "comment"}},
			Output: []string{"const value = 1;\n"},
		},
		{
			Code:   "/* a block */\nconst value = 1;\n",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "comment"}},
			Output: []string{"const value = 1;\n"},
		},
		{
			// Consecutive comment lines report once, as one run.
			Code:   "// first\n// second\nconst value = 1;\n",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "comment"}},
			Output: []string{"const value = 1;\n"},
		},
		{
			// A comment after the last statement is still trivia of the file.
			Code:   "const value = 1;\n// dangling\n",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "comment"}},
			Output: []string{"const value = 1;\n"},
		},
	})
}
