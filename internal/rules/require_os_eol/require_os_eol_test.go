package require_os_eol

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestRequireOsEol(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireOsEolRule, []rule_tester.ValidTestCase{
		{Code: "import { EOL } from \"node:os\";\nexport const joined = `a${EOL}b`;"},
		// A doubled backslash is a literal backslash followed by an n.
		{Code: `export const pattern = "a\\nb";`},
		{Code: `export const plain = "no newline here";`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   "export const joined = \"a\\nb\";",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "hardcodedNewline"}},
			Output: []string{"import { EOL } from \"node:os\";\nexport const joined = `a${EOL}b`;"},
		},
		{
			// A literal that is nothing but a newline becomes EOL itself.
			Code:   "export const newline = \"\\n\";",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "hardcodedNewline"}},
			Output: []string{"import { EOL } from \"node:os\";\nexport const newline = EOL;"},
		},
		{
			// An existing node:os import gains EOL rather than a second import.
			Code:   "import { tmpdir } from \"node:os\";\nexport const joined = tmpdir() + \"a\\nb\";",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "hardcodedNewline"}},
			Output: []string{"import { EOL, tmpdir } from \"node:os\";\nexport const joined = tmpdir() + `a${EOL}b`;"},
		},
		{
			Code:   "export const joined = `a\\nb`;",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "hardcodedNewline"}},
			Output: []string{"import { EOL } from \"node:os\";\nexport const joined = `a${EOL}b`;"},
		},
		{
			Code:   "export const carriage = \"a\\r\\nb\";",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "hardcodedNewline"}},
			Output: []string{"import { EOL } from \"node:os\";\nexport const carriage = `a${EOL}b`;"},
		},
		{
			// A template with a substitution keeps its holes; only the escape moves.
			Code:   "declare const n: number;\nexport const joined = `a\\n${n}b\\nc`;",
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "hardcodedNewline"}},
			Output: []string{"import { EOL } from \"node:os\";\ndeclare const n: number;\nexport const joined = `a${EOL}${n}b${EOL}c`;"},
		},
	})
}
