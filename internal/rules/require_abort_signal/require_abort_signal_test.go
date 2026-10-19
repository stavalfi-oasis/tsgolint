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

// kysely spells the same `execute` on a query builder, on a schema builder that
// takes nothing, and on a transaction that takes the options second.
const kyselyStub = `
interface AbortSignal {}
declare const signal: AbortSignal;
interface AbortableQueryOptions { signal?: AbortSignal }
declare const query: {
  execute(options?: AbortableQueryOptions): Promise<unknown[]>;
  executeTakeFirst(options?: AbortableQueryOptions): Promise<unknown>;
  executeTakeFirstOrThrow(options?: AbortableQueryOptions): Promise<unknown>;
};
declare const db: {
  executeQuery(compiled: unknown, options?: AbortableQueryOptions): Promise<unknown>;
  transaction(): { execute<T>(callback: () => Promise<T>, options?: AbortableQueryOptions): Promise<T> };
};
declare const schema: { execute(): Promise<void> };
`

// node's fs/promises readFile: the options slot takes an encoding string or an
// options object, and only the object can carry the signal.
const readFileStub = `
interface AbortSignal {}
declare const signal: AbortSignal;
declare const options: { encoding: "utf8"; signal: AbortSignal };
declare function readFile(path: string, options?: "utf8" | null | { encoding?: "utf8"; signal?: AbortSignal }): Promise<string>;
`

// The rule carries no built-in list, so every case states the calls it watches.
var options = RequireAbortSignalOptions{Calls: []CancellableCall{
	{Name: "fetch", OptionName: "signal", OptionsIndex: 1},
	{Name: "on", OptionName: "signal", OptionsIndex: 2},
	{Name: "send", OptionName: "abortSignal", OptionsIndex: 1},
	{Name: "readFile", OptionName: "signal", OptionsIndex: 1},
	{Name: "execute", OptionName: "signal", OptionsIndex: 0, RequireDeclaredOption: true},
	{Name: "execute", OptionName: "signal", OptionsIndex: 1, RequireDeclaredOption: true},
	{Name: "executeQuery", OptionName: "signal", OptionsIndex: 1, RequireDeclaredOption: true},
	{Name: "executeTakeFirst", OptionName: "signal", OptionsIndex: 0, RequireDeclaredOption: true},
	{Name: "executeTakeFirstOrThrow", OptionName: "signal", OptionsIndex: 0, RequireDeclaredOption: true},
}}

func TestRequireAbortSignal(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &RequireAbortSignalRule, []rule_tester.ValidTestCase{
		{Options: options, Code: stub + `fetch("https://x", { signal });`},
		{Options: options, Code: stub + `
declare const options: { signal?: AbortSignal };
fetch("https://x", options);
    `},
		// Same method name, but the signature has no signal option. The JS rule
		// matched on the name alone and reported this.
		{Options: options, Code: `
declare const queue: { send(body: string, options?: { retries?: number }): void };
queue.send("body", { retries: 2 });
    `},
		{Options: options, Code: `
declare const emitter: { on(event: string, handler: () => void): void };
emitter.on("data", () => {});
    `},
		{Options: options, Code: kyselyStub + `query.execute({ signal });`},
		{Options: options, Code: kyselyStub + `query.executeTakeFirst({ signal });`},
		{Options: options, Code: kyselyStub + `query.executeTakeFirstOrThrow({ signal });`},
		{Options: options, Code: kyselyStub + `db.executeQuery({}, { signal });`},
		{Options: options, Code: kyselyStub + `db.transaction().execute(async () => 1, { signal });`},
		// A signature that declares no options at all is a different `execute`.
		{Options: options, Code: kyselyStub + `schema.execute();`},
		{Options: options, Code: readFileStub + `readFile("a.txt", { encoding: "utf8", signal });`},
		{Options: options, Code: readFileStub + `readFile("a.txt", options);`},
	}, []rule_tester.InvalidTestCase{
		{
			Options: options,
			Code: stub + `fetch("https://x");`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireAbortSignal", Line: 5, Column: 1, EndLine: 5, EndColumn: 19},
			},
		},
		{
			Options: options,
			Code: stub + `fetch("https://x", {});`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireAbortSignal", Line: 5, Column: 1, EndLine: 5, EndColumn: 23},
			},
		},
		{
			Options: options,
			Code: kyselyStub + `query.executeTakeFirst();`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireAbortSignal", Line: 15, Column: 1, EndLine: 15, EndColumn: 25},
			},
		},
		{
			Options: options,
			Code: kyselyStub + `query.execute({});`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireAbortSignal", Line: 15, Column: 1, EndLine: 15, EndColumn: 18},
			},
		},
		{
			Options: options,
			Code: kyselyStub + `db.executeQuery({});`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireAbortSignal", Line: 15, Column: 1, EndLine: 15, EndColumn: 20},
			},
		},
		{
			Options: options,
			Code: kyselyStub + `db.transaction().execute(async () => 1);`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireAbortSignal", Line: 15, Column: 1, EndLine: 15, EndColumn: 40},
			},
		},
		{
			Options: options,
			Code:    readFileStub + `readFile("a.txt", "utf8");`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireAbortSignal", Line: 6, Column: 1, EndLine: 6, EndColumn: 26},
			},
		},
		{
			Options: options,
			Code:    readFileStub + `readFile("a.txt", null);`,
			Errors: []rule_tester.InvalidTestCaseError{
				{MessageId: "requireAbortSignal", Line: 6, Column: 1, EndLine: 6, EndColumn: 24},
			},
		},
	})
}
