package require_abort_signal_in_while

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestRequireAbortSignalInWhile(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireAbortSignalInWhileRule, []rule_tester.ValidTestCase{
		{Code: `declare const signal: AbortSignal; while (!signal.aborted) {}`},
		// The check only has to appear in the condition; the loop keeps its own
		// stop conditions next to it.
		{Code: `declare const signal: AbortSignal; declare let taken: number; while (!signal.aborted && taken < 10) { taken += 1; }`},
		{Code: `declare const signal: AbortSignal; declare let taken: number; while (taken < 10 && !signal.aborted) { taken += 1; }`},
		// Any receiver, however deep the chain.
		{Code: `declare const controller: AbortController; while (!controller.signal.aborted) {}`},
		{Code: `declare const signal: AbortSignal; while (!(signal.aborted)) {}`},
		{Code: `declare const signal: AbortSignal; while (!signal["aborted"]) {}`},
		// A union still carries the signal on every branch that has one.
		{Code: `declare const signal: AbortSignal | undefined; while (!signal?.aborted) {}`},
		// A subclass is still an AbortSignal.
		{Code: `declare class TimeoutSignal extends AbortSignal {}; declare const signal: TimeoutSignal; while (!signal.aborted) {}`},
		{Code: `declare const signal: AbortSignal; do {} while (!signal.aborted);`},
		// A `for` is a different statement and is not this rule's business.
		{Code: `for (let i = 0; i < 10; i += 1) {}`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `declare let taken: number; while (taken < 10) { taken += 1; }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireAbortSignalInWhile", Line: 1, Column: 28}},
		},
		{
			Code:   `while (true) { break; }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireAbortSignalInWhile", Line: 1, Column: 1}},
		},
		{
			// The inverse test never ends the loop on abort, so the bare member
			// access does not count.
			Code:   `declare const signal: AbortSignal; while (signal.aborted) {}`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireAbortSignalInWhile", Line: 1, Column: 36}},
		},
		{
			// An unrelated object carrying an `aborted` flag is not a signal —
			// this is what the name check alone got wrong.
			Code:   `declare const state: { aborted: boolean }; while (!state.aborted) {}`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireAbortSignalInWhile", Line: 1, Column: 44}},
		},
		{
			// Same shape, same property name, still not an AbortSignal.
			Code:   `declare class Job { aborted = false }; declare const job: Job; while (!job.aborted) {}`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireAbortSignalInWhile", Line: 1, Column: 64}},
		},
		{
			// A property that merely reads like the signal is not the signal.
			Code:   `declare const state: { abort: boolean }; while (!state.abort) {}`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireAbortSignalInWhile", Line: 1, Column: 42}},
		},
		{
			Code:   `declare let draining: boolean; do { draining = false; } while (draining);`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireAbortSignalInWhile", Line: 1, Column: 32}},
		},
	})
}
