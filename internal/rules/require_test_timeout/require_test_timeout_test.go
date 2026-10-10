package require_test_timeout

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestRequireTestTimeout(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireTestTimeoutRule, []rule_tester.ValidTestCase{
		{Code: `it('works', { timeout: 5_000 }, async () => {});`},
		{Code: `it('works', { timeout }, async () => {});`},
		{Code: `it('works', { concurrency: 1, timeout: 1 }, () => {});`},
		{Code: `it.only('works', { timeout: 1 }, () => {});`},
		{Code: `it.skip('works', { timeout: 1 }, () => {});`},
		// A spread may carry `timeout`, and a non-literal options argument is
		// unknowable without following the binding.
		{Code: `it('works', { ...options }, () => {});`},
		{Code: `it('works', options, () => {});`},
		// `describe` is exempt, with or without options.
		{Code: `describe('group', () => {});`},
		{Code: `describe('group', { timeout: 1 }, () => {});`},
		// No body to time out.
		{Code: `it.todo('later');`},
		// Not the test runner's `it`.
		{Code: `const it = (name: string) => name; it('works');`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   `it('works', async () => {});`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireTestTimeout", Line: 1, Column: 1, EndColumn: 3}},
		},
		{
			Code:   `it('works', function () {});`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireTestTimeout", Line: 1, Column: 1, EndColumn: 3}},
		},
		{
			Code:   `it('works', {}, () => {});`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireTestTimeout", Line: 1, Column: 1, EndColumn: 3}},
		},
		{
			Code:   `it('works', { concurrency: 1 }, () => {});`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireTestTimeout", Line: 1, Column: 1, EndColumn: 3}},
		},
		{
			Code:   `it('works');`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireTestTimeout", Line: 1, Column: 1, EndColumn: 3}},
		},
		{
			Code:   `it.only('works', () => {});`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireTestTimeout", Line: 1, Column: 1, EndColumn: 8}},
		},
		{
			Code:   `it.skip('works', () => {});`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireTestTimeout", Line: 1, Column: 1, EndColumn: 8}},
		},
		{
			Code: `
import { describe, it } from 'node:test';
describe('group', () => {
  it('works', async () => {});
});
			`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "requireTestTimeout", Line: 4, Column: 3, EndColumn: 5}},
		},
	})
}
