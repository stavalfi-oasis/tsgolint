package no_curl

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

const stub = `
declare function execFile(command: string, args?: string[]): void;
declare function spawn(command: string, args?: string[]): void;
declare const child: { execFile(command: string, args?: string[]): void };
`

var options = NoCurlOptions{
	ExecFunctions: []string{"execFile", "spawn"},
	Groups: []BannedCommandGroup{
		{Commands: []string{"curl", "wget"}, Message: "Shelling out to curl/wget is banned."},
	},
}

func TestNoCurl(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoCurlRule, []rule_tester.ValidTestCase{
		{Code: stub + `execFile("git", ["status"]);`, Options: options},
		{Code: stub + `declare function other(command: string): void; other("curl");`, Options: options},
	}, []rule_tester.InvalidTestCase{
		{
			Code:    stub + `execFile("curl", ["https://example.com"]);`,
			Options: options,
			Errors:  []rule_tester.InvalidTestCaseError{{MessageId: "bannedCommand"}},
		},
		{
			Code:    stub + `spawn("wget");`,
			Options: options,
			Errors:  []rule_tester.InvalidTestCaseError{{MessageId: "bannedCommand"}},
		},
		{
			Code:    stub + `child.execFile("curl");`,
			Options: options,
			Errors:  []rule_tester.InvalidTestCaseError{{MessageId: "bannedCommand"}},
		},
	})
}
