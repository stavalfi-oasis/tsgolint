package zod_schemas_file_only

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

// zod's schema classes are named ZodSomething, which is what the rule keys off.
const zodStub = `
declare class ZodString {
  optional(): ZodString;
}
declare class ZodObject {}
declare const z: { string(): ZodString; object(): ZodObject };
`

// The rule tester lints a file that is not named zod-schemas.ts, so these cover
// the "declared in the wrong file" half. The duplicate-file half needs several
// real files in one program and is covered by the poc integration test.
func TestZodSchemasFileOnly(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &ZodSchemasFileOnlyRule, []rule_tester.ValidTestCase{
		{Code: `
const port = 8080;
    `},
		// Importing a schema is the whole point — only declaring one here is banned.
		{Code: zodStub + `
declare const imported: ZodString;
const used = imported.optional() ? 1 : 2;
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: zodStub + `const schema = z.string();`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "schemaOutsideSchemasFile", Line: 7, Column: 7, EndLine: 7, EndColumn: 13},
			},
		},
		{
			Code: zodStub + `export const body = z.object();`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "schemaOutsideSchemasFile", Line: 7, Column: 14, EndLine: 7, EndColumn: 18},
			},
		},
		// Built through a helper rather than a `z.` chain — a syntactic rule keyed
		// on `z.` would miss this; the declared type still says ZodString.
		{
			Code: zodStub + `
function make(): ZodString { return z.string(); }
const viaHelper = make();
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "schemaOutsideSchemasFile", Line: 9, Column: 7, EndLine: 9, EndColumn: 16},
			},
		},
	})
}

// subPackageDirs makes a matching directory its own package, so a command folder
// can hold its own zod-schemas.ts next to the package-level one.
func TestPackageDirOfSubPackageDirs(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tsconfig.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := filepath.Join(root, "src", "commands", "start-backends")
	cases := []struct {
		file           string
		subPackageDirs []string
		want           string
	}{
		{filepath.Join(root, "src", "zod-schemas.ts"), []string{"src/commands/*"}, root},
		{filepath.Join(command, "zod-schemas.ts"), nil, root},
		{filepath.Join(command, "zod-schemas.ts"), []string{"src/commands/*"}, command},
		{filepath.Join(command, "nested", "zod-schemas.ts"), []string{"src/commands/*"}, command},
		{filepath.Join(command, "zod-schemas.ts"), []string{"src/other/*"}, root},
	}
	for _, c := range cases {
		if got := packageDirOf(c.file, c.subPackageDirs); got != c.want {
			t.Errorf("packageDirOf(%q, %v) = %q, want %q", c.file, c.subPackageDirs, got, c.want)
		}
	}
}
