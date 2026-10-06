package require_abort_signal_in_while

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestRequireAbortSignalInWhile(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireAbortSignalInWhileRule, []rule_tester.ValidTestCase{
		{Code: `declare const signal: { aborted: boolean }; while (!signal.aborted) {}`},
		// The check only has to appear in the condition; the loop keeps its own
		// stop conditions next to it.
		{Code: `declare const signal: { aborted: boolean }; declare let taken: number; while (!signal.aborted && taken < 10) { taken += 1; }`},
		{Code: `declare const signal: { aborted: boolean }; declare let taken: number; while (taken < 10 && !signal.aborted) { taken += 1; }`},
		// Any receiver, however deep the chain.
		{Code: `declare const self: { controller: { signal: { aborted: boolean } } }; while (!self.controller.signal.aborted) {}`},
		{Code: `declare const signal: { aborted: boolean }; while (!(signal.aborted)) {}`},
		{Code: `declare const signal: { aborted: boolean }; while (!signal?.aborted) {}`},
		{Code: `declare const signal: { aborted: boolean }; while (!signal["aborted"]) {}`},
		{Code: `declare const signal: { aborted: boolean }; do {} while (!signal.aborted);`},
		// A `for` is a different statement and stays this rule's business.
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
			Code:   `declare const signal: { aborted: boolean }; while (signal.aborted) {}`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireAbortSignalInWhile", Line: 1, Column: 45}},
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
