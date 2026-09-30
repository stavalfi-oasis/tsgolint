package class_name_matches_filename

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestClassNameMatchesFilename(t *testing.T) {
	// Not parallel: each case compiles under a different file name, and the
	// fixtures tsconfig globs the whole directory.
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &ClassNameMatchesFilenameRule, []rule_tester.ValidTestCase{
		{FileName: "gh-gateway.service.ts", Code: `export class GhGatewayService {}`},
		// index.ts names nothing in particular.
		{FileName: "index.ts", Code: `export class Anything {}`},
		// Only the file's top-level class is pinned to the filename.
		{FileName: "runner.ts", Code: `export class Runner { public static run(): void { class Inner {} new Inner(); } }`},
	}, []rule_tester.InvalidTestCase{
		{
			FileName: "gh-gateway.service.ts",
			Code:     `export class Gateway {}`,
			Errors:   []rule_tester.InvalidTestCaseError{{MessageId: "classNameMismatch"}},
			Output:   []string{`export class GhGatewayService {}`},
		},
		{
			// Every reference to the class is renamed with it.
			FileName: "runner.ts",
			Code:     `class Other {}` + "\n" + `export const made = new Other();`,
			Errors:   []rule_tester.InvalidTestCaseError{{MessageId: "classNameMismatch"}},
			Output:   []string{`class Runner {}` + "\n" + `export const made = new Runner();`},
		},
	})
}
