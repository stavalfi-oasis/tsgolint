package no_banned_words

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

var options = NoBannedWordsOptions{
	Groups: []BannedGroup{
		{Message: "say 'Oasis Platform', not 'app'.", Words: []string{"app", "product"}},
	},
}

func TestNoBannedWords(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoBannedWordsRule, []rule_tester.ValidTestCase{
		{Code: `const label = "Oasis Platform";`, Options: options},
		// Whole words only: "apple" is not "app".
		{Code: `const label = "apple pie";`, Options: options},
		{Code: `const label = "nothing here";`, Options: options},
	}, []rule_tester.InvalidTestCase{
		{
			Code:    `const label = "open the app now";`,
			Options: options,
			Errors:  []rule_tester.InvalidTestCaseError{{MessageId: "bannedWord"}},
		},
		{
			Code:    `const label = "the APP is down";`,
			Options: options,
			Errors:  []rule_tester.InvalidTestCaseError{{MessageId: "bannedWord"}},
		},
		{
			Code:    "declare const n: number; const label = `the product ${n}`;",
			Options: options,
			Errors:  []rule_tester.InvalidTestCaseError{{MessageId: "bannedWord"}},
		},
	})
}
