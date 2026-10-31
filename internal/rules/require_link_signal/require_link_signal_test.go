package require_link_signal

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

// The fixtures project sets `"types": []`, so the base class is declared here.
const stub = `
interface AbortSignal { aborted: boolean }
declare function fetch(url: string, init?: { signal?: AbortSignal }): Promise<unknown>;
declare abstract class Base {
  protected linkSignal(options: { signal: AbortSignal }): AbortSignal;
}
`

func TestRequireLinkSignal(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireLinkSignalRule, []rule_tester.ValidTestCase{
		{Code: stub + `
class A extends Base {
  run({ signal }: { signal: AbortSignal }) {
    const linked = super.linkSignal({ signal });
    return fetch("x", { signal: linked });
  }
}`},
		{Code: stub + `
class A extends Base {
  run(options: { signal: AbortSignal }) {
    return fetch("x", { signal: super.linkSignal({ signal: options.signal }) });
  }
}`},
		// No `linkSignal` to reach: the class does not inherit it.
		{Code: stub + `
class B {}
class A extends B {
  run({ signal }: { signal: AbortSignal }) { return fetch("x", { signal }); }
}`},
		{Code: stub + `
function run({ signal }: { signal: AbortSignal }) { return fetch("x", { signal }); }
`},
	}, []rule_tester.InvalidTestCase{
		{
			Code: stub + `
class A extends Base {
  run({ signal }: { signal: AbortSignal }) { return fetch("x", { signal }); }
}`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireLinkSignal", Line: 9, Column: 66, EndLine: 9, EndColumn: 72},
			},
		},
		{
			Code: stub + `
class A extends Base {
  run({ signal: renamed }: { signal: AbortSignal }) { return fetch("x", { signal: renamed }); }
}`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireLinkSignal", Line: 9, Column: 83, EndLine: 9, EndColumn: 90},
			},
		},
		{
			Code: stub + `
class A extends Base {
  run(abort: AbortSignal) { const alias = abort; return fetch("x", { signal: alias }); }
}`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireLinkSignal", Line: 9, Column: 43, EndLine: 9, EndColumn: 48},
			},
		},
		{
			Code: stub + `
class A extends Base {
  run(options: { signal: AbortSignal }) { return fetch("x", { signal: options.signal }); }
}`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireLinkSignal", Line: 9, Column: 71, EndLine: 9, EndColumn: 85},
			},
		},
	})
}
