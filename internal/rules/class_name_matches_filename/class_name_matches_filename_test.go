package class_name_matches_filename

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

// Every file name this test compiles under. The fixtures tsconfig globs the
// directory and the base VFS is cached across cases, so a name introduced by one
// case can still be in the program's file list for the next. Declaring them all
// in every case keeps that list satisfiable whichever case runs.
var fileNames = []string{"gh-gateway.service.ts", "index.ts", "runner.ts"}

func others(fileName string) map[string]string {
	files := map[string]string{}
	for _, other := range fileNames {
		if other != fileName {
			files[other] = "export {};"
		}
	}
	return files
}

func TestClassNameMatchesFilename(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &ClassNameMatchesFilenameRule, []rule_tester.ValidTestCase{
		{
			FileName: "gh-gateway.service.ts",
			Files:    others("gh-gateway.service.ts"),
			Code:     `export class GhGatewayService {}`,
		},
		{
			// index.ts names nothing in particular.
			FileName: "index.ts",
			Files:    others("index.ts"),
			Code:     `export class Anything {}`,
		},
		{
			// Only the file's top-level class is pinned to the filename.
			FileName: "runner.ts",
			Files:    others("runner.ts"),
			Code:     `export class Runner { public static run(): void { class Inner {} new Inner(); } }`,
		},
	}, []rule_tester.InvalidTestCase{
		{
			FileName: "gh-gateway.service.ts",
			Files:    others("gh-gateway.service.ts"),
			Code:     `export class Gateway {}`,
			Errors:   []rule_tester.InvalidTestCaseError{{MessageId: "classNameMismatch"}},
			Output:   []string{`export class GhGatewayService {}`},
		},
		{
			// Every reference to the class is renamed with it.
			FileName: "runner.ts",
			Files:    others("runner.ts"),
			Code:     "class Other {}\nexport const made = new Other();",
			Errors:   []rule_tester.InvalidTestCaseError{{MessageId: "classNameMismatch"}},
			Output:   []string{"class Runner {}\nexport const made = new Runner();"},
		},
	})
}
