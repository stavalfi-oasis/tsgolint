package require_abort_signal

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

// The fixtures project sets `"types": []`, so these declare the shapes locally.
// What matters to the rule is the signature, not where it came from.
const stub = `
interface AbortSignal {}
declare function fetch(url: string, init?: { signal?: AbortSignal }): Promise<unknown>;
declare const signal: AbortSignal;
`

func TestRequireAbortSignal(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireAbortSignalRule, []rule_tester.ValidTestCase{
		{Code: stub + `fetch("https://x", { signal });`},
		{Code: stub + `
declare const options: { signal?: AbortSignal };
fetch("https://x", options);
    `},
		// Same method name, but the signature has no signal option. The JS rule
		// matched on the name alone and reported this.
		{Code: `
declare const queue: { send(body: string, options?: { retries?: number }): void };
queue.send("body", { retries: 2 });
    `},
		{Code: `
declare const emitter: { on(event: string, handler: () => void): void };
emitter.on("data", () => {});
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: stub + `fetch("https://x");`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireAbortSignal", Line: 5, Column: 1, EndLine: 5, EndColumn: 19},
			},
		},
		{
			Code: stub + `fetch("https://x", {});`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireAbortSignal", Line: 5, Column: 1, EndLine: 5, EndColumn: 23},
			},
		},
	})
}
