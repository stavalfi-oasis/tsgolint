package prefer_await_to_callbacks

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

const stub = `
declare class Hono { onError(handler: (error: Error, c: unknown) => void): void }
declare class Socket { onError(handler: (error: Error) => void): void }
declare const app: Hono;
declare const maybeApp: Hono | undefined;
declare const socket: Socket;
declare function heart(...args: unknown[]): void;
`

var options = PreferAwaitToCallbacksOptions{
	Allowed: []AllowedMethod{{Class: "Hono", Method: "onError"}},
}

func TestPreferAwaitToCallbacks(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &PreferAwaitToCallbacksRule, []rule_tester.ValidTestCase{
		{Code: stub + `app.onError((error, c) => {});`, Options: options},
		{Code: stub + `maybeApp?.onError((error) => {});`, Options: options},
		{Code: stub + `declare const dbConn: { on(name: string, h: (err: Error) => void): void }; dbConn.on("error", err => {});`},
		{Code: stub + `heart(something => {});`},
		{Code: `declare const errors: Error[]; errors.filter(err => err.message === "x");`},
		{Code: stub + `async function hi() { await heart(err => {}); }`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:    stub + `socket.onError((error) => {});`,
			Options: options,
			Errors:  []rule_tester.InvalidTestCaseError{{MessageId: "preferAwaitToCallbacks"}},
		},
		{
			Code:   stub + `app.onError((error, c) => {});`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferAwaitToCallbacks"}},
		},
		{
			Code:   stub + `heart(err => {});`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferAwaitToCallbacks"}},
		},
		{
			Code:   `function getData(id: string, callback: () => void) {}`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferAwaitToCallbacks"}},
		},
		{
			Code:   `declare const cb: () => void; cb();`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferAwaitToCallbacks"}},
		},
	})
}
