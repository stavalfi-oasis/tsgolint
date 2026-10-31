package require_abort_rethrow

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

const header = `import { isAbortError } from "@poc/libs/utils.ts";
declare function work(): void;
`

func TestRequireAbortRethrow(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireAbortRethrowRule, []rule_tester.ValidTestCase{
		{Code: header + `
try {
  work();
} catch (error: unknown) {
  if (isAbortError(error)) {
    throw error;
  }
  work();
}`},
		{Code: header + `
try { work(); } catch (e) { if (isAbortError(e)) throw e; }`},
		// Swallowing aborts on purpose is still a decision about them.
		{Code: header + `
try { work(); } catch (e) { if (!isAbortError(e)) { throw e; } }`},
		// An aliased import is still the guard.
		{Code: `import { isAbortError as aborted } from "@poc/libs/utils.ts";
try { work(); } catch (e) { if (aborted(e)) { throw e; } }`},
	}, []rule_tester.InvalidTestCase{
		{
			Code: header + `
function f() {
  try {
    work();
  } catch (error: unknown) {
    work();
  }
}`,
			Output: []string{header + `
function f() {
  try {
    work();
  } catch (error: unknown) {
    if (isAbortError(error)) {
      throw error;
    }
    work();
  }
}`},
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingAbortRethrow", Line: 7, Column: 5, EndLine: 9, EndColumn: 4},
			},
		},
		// Not first: something can swallow the abort before the check.
		{
			Code: header + `
try {
  work();
} catch (e) {
  work();
  if (isAbortError(e)) {
    throw e;
  }
}`,
			Output: []string{header + `
try {
  work();
} catch (e) {
  if (isAbortError(e)) {
    throw e;
  }
  work();
  if (isAbortError(e)) {
    throw e;
  }
}`},
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingAbortRethrow", Line: 6, Column: 3, EndLine: 11, EndColumn: 2},
			},
		},
		// No binding, empty body, guard not imported yet.
		{
			Code: `declare function work(): void;
try {
  work();
} catch {}`,
			Output: []string{`import { isAbortError } from "@poc/libs/utils.ts";
declare function work(): void;
try {
  work();
} catch (error) {
  if (isAbortError(error)) {
    throw error;
  }
}`},
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingAbortRethrow", Line: 4, Column: 3, EndLine: 4, EndColumn: 11},
			},
		},
		// An else branch can still handle the abort.
		{
			Code: header + `
try { work(); } catch (e) { if (isAbortError(e)) { throw e; } else { work(); } }`,
			Output: []string{header + `
try { work(); } catch (e) {
  if (isAbortError(e)) {
    throw e;
  }
  if (isAbortError(e)) { throw e; } else { work(); } }`},
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingAbortRethrow", Line: 4, Column: 17, EndLine: 4, EndColumn: 81},
			},
		},
		// Destructured: reported, not fixed.
		{
			Code: header + `
try { work(); } catch ({ message }) { work(); }`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "missingAbortRethrow", Line: 4, Column: 17, EndLine: 4, EndColumn: 48},
			},
		},
	})
}
