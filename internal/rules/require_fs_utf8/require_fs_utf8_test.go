package require_fs_utf8

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

// The fixtures project sets `"types": []` and has no node_modules, so the
// positive cases — which need a real `fs` package for the specifier to resolve —
// are covered by the poc integration test instead. What is pinned here is the
// half the JS rule got wrong: it matched on the method name alone, so any
// `readFile`/`writeFile` on any object was reported.
func TestRequireFsUtf8(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireFsUtf8Rule, []rule_tester.ValidTestCase{
		{Code: `
declare const storage: { readFile(path: string): Promise<string> };
storage.readFile("a.txt");
    `},
		{Code: `
declare const bucket: { writeFileSync(path: string, body: string): void };
bucket.writeFileSync("a.txt", "body");
    `},
		{Code: `
function readFile(path: string): string { return path; }
readFile("a.txt");
    `},
	}, []rule_tester.InvalidTestCase{})
}
