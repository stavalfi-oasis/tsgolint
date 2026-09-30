package no_process_stream_write

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

// The fixtures project sets `"types": []`, so node's globals are not available.
// This mirrors the shape the rule keys off: a value whose type symbol is named
// `Process`.
const processStub = `
interface WriteStream { write(chunk: string): boolean }
interface Process { stdout: WriteStream; stderr: WriteStream }
declare const process: Process;
`

func TestNoProcessStreamWrite(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoProcessStreamWriteRule, []rule_tester.ValidTestCase{
		{Code: processStub + `process.stdout.end();`},
		// The JS rule matched the identifier `process` by name, so this local
		// stand-in was a false positive.
		{Code: `
const process = { stdout: { write(chunk: string): void {} } };
process.stdout.write("x");
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: processStub + `process.stdout.write("x");`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "streamWrite", Line: 5, Column: 1, EndLine: 5, EndColumn: 26},
			},
			Output: []string{processStub + `console.log("x");`},
		},
		{
			Code: processStub + `process.stderr.write("x");`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "streamWrite", Line: 5, Column: 1, EndLine: 5, EndColumn: 26},
			},
			Output: []string{processStub + `console.error("x");`},
		},
	})
}
