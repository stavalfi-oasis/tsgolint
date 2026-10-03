package no_client_response_json

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

const clientResponseDeclaration = `
interface ClientResponse<T> {
  json(): Promise<T>;
}
declare const typed: ClientResponse<{ bytes: number }>;
declare const plain: Response;
declare const parser: { json(): Promise<unknown> };
`

func TestNoClientResponseJson(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoClientResponseJsonRule, []rule_tester.ValidTestCase{
		{Code: clientResponseDeclaration + `const a = plain.json();`},
		{Code: clientResponseDeclaration + `const a = parser.json();`},
		{Code: clientResponseDeclaration + `const a = typed.text();`},
		{Code: clientResponseDeclaration + `const a = JSON.parse("{}");`},
	}, []rule_tester.InvalidTestCase{
		{
			Code:   clientResponseDeclaration + `const a = typed.json();`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "clientResponseJson"}},
		},
		{
			Code:   clientResponseDeclaration + `async function read() { return await typed.json(); }`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "clientResponseJson"}},
		},
		{
			Code: clientResponseDeclaration + `
declare const maybe: Response | ClientResponse<{ bytes: number }>;
const a = maybe.json();`,
			Errors: []rule_tester.InvalidTestCaseError{{MessageId: "clientResponseJson"}},
		},
	})
}
