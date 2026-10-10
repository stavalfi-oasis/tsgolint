package no_primitive_collection_key

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoPrimitiveCollectionKey(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoPrimitiveCollectionKeyRule, []rule_tester.ValidTestCase{
		// A named alias is the whole point of the rule.
		{Code: `
type TenantId = string;
interface Session {}
declare const sessions: Map<TenantId, Session>;
    `},
		// The value position is nobody's business here.
		{Code: `
type TenantId = string;
declare const names: Map<TenantId, string>;
    `},
		// A literal union already names every key it accepts.
		{Code: `
declare const modes: Map<'read' | 'write', number>;
    `},
		{Code: `
declare const mode: Map<'read', number>;
    `},
		// The caller names a type parameter, not this file.
		{Code: `
declare function index<Key, Value>(entries: Map<Key, Value>): void;
    `},
		{Code: `
class Cache<Key> {
  private readonly entries: Map<Key, string> = new Map<Key, string>();
}
    `},
		{Code: `
type FindingSlug = string;
declare const slugs: Set<FindingSlug>;
    `},
		{Code: `
type TenantId = string;
declare const counts: Record<TenantId, number>;
    `},
		{Code: `
type TenantId = string;
type AccountId = string;
declare const owners: Map<TenantId | AccountId, string>;
    `},
		// Not one of the containers the rule knows about.
		{Code: `
declare const names: Array<string>;
declare const pending: Promise<string>;
    `},
		{Code: `
type TenantId = string;
interface Session {}
const sessions = new Map<TenantId, Session>();
sessions.clear();
    `},
		// The annotation names the key; the `new` side has nothing left to say.
		{Code: `
type TenantId = string;
interface Session {}
const sessions: Map<TenantId, Session> = new Map();
sessions.clear();
    `},
		// A qualified alias names the key just as well.
		{Code: `
declare namespace ids {
  type TenantId = string;
}
declare const sessions: Map<ids.TenantId, number>;
    `},
	}, []rule_tester.InvalidTestCase{
		{
			Code: `
interface Session {}
declare const sessions: Map<string, Session>;
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "primitiveKey"}},
		},
		{
			Code: `
declare const counts: Map<number, string>;
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "primitiveKey"}},
		},
		{
			Code: `
declare const flags: ReadonlySet<boolean>;
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "primitiveKey"}},
		},
		{
			Code: `
declare const slugs: Set<string>;
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "primitiveKey"}},
		},
		// `Record` is a map written another way.
		{
			Code: `
declare const payload: Record<string, unknown>;
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "primitiveKey"}},
		},
		// An alias that renames `string` without naming it.
		{
			Code: `
type Str = string;
declare const sessions: Map<Str, number>;
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "meaninglessAlias"}},
		},
		{
			Code: `
type Value = string;
declare const sessions: Map<Value, number>;
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "meaninglessAlias"}},
		},
		// An inline brand leaves the key unnamed at the use site.
		{
			Code: `
declare const sessions: Map<string & { readonly __brand: 'TenantId' }, number>;
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "primitiveKey"}},
		},
		// A union is only named when every constituent is.
		{
			Code: `
type TenantId = string;
declare const owners: Map<TenantId | string, number>;
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "primitiveKey"}},
		},
		{
			Code: `
declare const sessions: Map<(string), number>;
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "primitiveKey"}},
		},
		{
			Code: `
const sessions = new Map<string, number>();
sessions.clear();
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "primitiveKey"}},
		},
		// Inference names nothing, so the key type is unreadable.
		{
			Code: `
const sessions = new Map();
sessions.clear();
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingTypeArguments"}},
		},
		{
			Code: `
class Cache {
  private readonly entries = new Set();
}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "missingTypeArguments"}},
		},
		{
			Code: `
interface Session {}
class Sessions extends Map<string, Session> {}
      `,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "primitiveKey"}},
		},
		// Nested containers are each checked on their own.
		{
			Code: `
declare const nested: Map<string, Set<number>>;
      `,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "primitiveKey"},
				{MessageId: "primitiveKey"},
			},
		},
	})
}
