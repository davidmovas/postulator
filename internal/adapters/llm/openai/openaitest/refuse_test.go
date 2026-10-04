package openaitest_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/openai/openaitest"
)

const strictSchema = `{"type":"object","properties":{"title":{"type":"string"},"note":{"type":["string","null"]},` +
	`"parts":{"type":"array","items":{"type":"object","properties":{"h":{"type":"string"}},"required":["h"],"additionalProperties":false}}},` +
	`"required":["title","note","parts"],"additionalProperties":false}`

func TestTheServerRefusesWhatTheProbeSawRefused(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		body    string
		status  int
		param   any
		code    any
		message string
	}{
		{
			name:   "a request without a model",
			body:   `{"input":[{"role":"user","content":"hi"}]}`,
			status: http.StatusBadRequest, param: "model", code: "missing_required_parameter",
			message: "Missing required parameter: 'model'.",
		},
		{
			name:   "minimal effort",
			body:   `{"model":"gpt-5.6-luna","input":"hi","reasoning":{"effort":"minimal"}}`,
			status: http.StatusBadRequest, param: "reasoning.effort", code: "unsupported_value",
			message: "Unsupported value: 'minimal' is not supported with the 'gpt-5.6-luna' model.",
		},
		{
			name:   "a temperature at the default effort",
			body:   `{"model":"gpt-5.6-terra","input":"hi","temperature":0.2}`,
			status: http.StatusBadRequest, param: "temperature",
			message: "Unsupported parameter: 'temperature' is not supported with this model.",
		},
		{
			name:   "a top p at medium effort",
			body:   `{"model":"gpt-5.6-terra","input":"hi","top_p":0.5,"reasoning":{"effort":"medium"}}`,
			status: http.StatusBadRequest, param: "top_p",
			message: "Unsupported parameter: 'top_p' is not supported with this model.",
		},
		{
			name:   "an unknown service tier",
			body:   `{"model":"gpt-5.6-terra","input":"hi","service_tier":"bogus"}`,
			status: http.StatusBadRequest, param: "service_tier", code: "invalid_value",
			message: "Invalid value: 'bogus'.",
		},
		{
			name:   "an output ceiling under sixteen",
			body:   `{"model":"gpt-5.6-terra","input":"hi","max_output_tokens":15}`,
			status: http.StatusBadRequest, param: "max_output_tokens", code: "integer_below_min_value",
			message: "Expected a value >= 16, but got 15 instead.",
		},
		{
			name:   "a cache key over sixty four characters",
			body:   `{"model":"gpt-5.6-terra","input":"hi","prompt_cache_key":"` + strings.Repeat("k", 65) + `"}`,
			status: http.StatusBadRequest, param: "prompt_cache_key", code: "string_above_max_length",
			message: "Expected a string with maximum length 64, but got a string with length 65 instead.",
		},
		{
			name:   "a strict schema that allows more properties",
			body:   `{"model":"gpt-5.6-terra","input":"hi","text":{"format":{"type":"json_schema","name":"x","strict":true,"schema":{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]}}}}`,
			status: http.StatusBadRequest, param: "text.format.schema", code: "invalid_json_schema",
			message: "'additionalProperties' is required to be supplied and to be false.",
		},
		{
			name:   "a strict schema that leaves a property out of required",
			body:   `{"model":"gpt-5.6-terra","input":"hi","text":{"format":{"type":"json_schema","name":"x","strict":true,"schema":{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}},"required":["a"],"additionalProperties":false}}}}`,
			status: http.StatusBadRequest, param: "text.format.schema", code: "invalid_json_schema",
			message: "Missing 'b'.",
		},
		{
			name:   "a nested strict object that allows more properties",
			body:   `{"model":"gpt-5.6-terra","input":"hi","text":{"format":{"type":"json_schema","name":"x","strict":true,"schema":{"type":"object","properties":{"a":{"anyOf":[{"type":"object","properties":{}},{"type":"null"}]}},"required":["a"],"additionalProperties":false}}}}`,
			status: http.StatusBadRequest, param: "text.format.schema", code: "invalid_json_schema",
			message: "'additionalProperties' is required to be supplied and to be false.",
		},
		{
			name:   "a strict schema whose root is not an object",
			body:   `{"model":"gpt-5.6-terra","input":"hi","text":{"format":{"type":"json_schema","name":"x","strict":true,"schema":{"anyOf":[{"type":"string"}]}}}}`,
			status: http.StatusBadRequest, param: "text.format.schema", code: "invalid_json_schema",
			message: "schema must be a JSON Schema of 'type: \"object\"'",
		},
		{
			name:   "a strict tool with loose parameters",
			body:   `{"model":"gpt-5.6-terra","input":"hi","tools":[{"type":"function","name":"f","strict":true,"parameters":{"type":"object","properties":{"a":{"type":"string"}}}}]}`,
			status: http.StatusBadRequest, param: "tools[0].parameters", code: "invalid_function_parameters",
			message: "Invalid schema for function 'f'",
		},
		{
			name:   "a call without its output",
			body:   `{"model":"gpt-5.6-terra","input":[{"role":"user","content":"hi"},{"type":"function_call","call_id":"call_1","name":"f","arguments":"{}"}]}`,
			status: http.StatusBadRequest, param: "input",
			message: "No tool output found for function call call_1.",
		},
		{
			name:   "an output without its call",
			body:   `{"model":"gpt-5.6-terra","input":[{"role":"user","content":"hi"},{"type":"function_call_output","call_id":"call_1","output":"{}"}]}`,
			status: http.StatusBadRequest, param: "input",
			message: "No tool call found for function call output with call_id call_1.",
		},
		{
			name:   "a reasoning item without its encrypted content",
			body:   `{"model":"gpt-5.6-terra","store":false,"input":[{"type":"reasoning","id":"rs_123","summary":[]},{"role":"user","content":"hi"}]}`,
			status: http.StatusNotFound, param: "input",
			message: "Item with id 'rs_123' not found. Items are not persisted when `store` is set to false.",
		},
		{
			name:   "a deferred tool without tool search",
			body:   `{"model":"gpt-5.6-terra","input":"hi","tools":[{"type":"function","name":"f","strict":false,"defer_loading":true,"parameters":{"type":"object"}}]}`,
			status: http.StatusBadRequest, param: "tools.defer_loading",
			message: "Invalid Value: 'tools.defer_loading'. Deferred tools require tools.tool_search.",
		},
		{
			name: "a namespace of deferred tools without tool search",
			body: `{"model":"gpt-5.6-terra","input":"hi","tools":[{"type":"namespace","name":"pages","description":"The page map.",` +
				`"tools":[{"type":"function","name":"f","strict":false,"defer_loading":true,"parameters":{"type":"object"}}]}]}`,
			status: http.StatusBadRequest, param: "tools.defer_loading",
			message: "Deferred tools require tools.tool_search.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			server.Enqueue(openaitest.Text("never").Reply())

			got := post(t, server, "/responses", openaitest.DefaultKey, tc.body)
			if got.Status != tc.status {
				t.Fatalf("status = %d, want %d: %s", got.Status, tc.status, got.Body)
			}
			fault := errorOf(t, got.Body)
			if fault["param"] != tc.param || fault["code"] != tc.code {
				t.Errorf("fault = %v, want param %v and code %v", fault, tc.param, tc.code)
			}
			if message := text(t, fault["message"]); !strings.Contains(message, tc.message) {
				t.Errorf("message = %q, want it to say %q", message, tc.message)
			}
			if server.Pending() != 1 {
				t.Errorf("pending = %d, want the refusal served before the queue", server.Pending())
			}
		})
	}
}

func TestTheServerAcceptsWhatTheProbeSawAccepted(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
	}{
		{name: "a plain input", body: `{"model":"gpt-5.6-terra","input":"hi"}`},
		{name: "a temperature at effort none", body: `{"model":"gpt-5.6-terra","input":"hi","temperature":0.2,"reasoning":{"effort":"none"}}`},
		{name: "every effort the model knows", body: `{"model":"gpt-5.6-luna","input":"hi","reasoning":{"effort":"max"}}`},
		{name: "flex", body: `{"model":"gpt-5.6-terra","input":"hi","service_tier":"flex"}`},
		{name: "an output ceiling of sixteen", body: `{"model":"gpt-5.6-terra","input":"hi","max_output_tokens":16}`},
		{name: "a cache key of sixty four characters", body: `{"model":"gpt-5.6-terra","input":"hi","prompt_cache_key":"` + strings.Repeat("k", 64) + `"}`},
		{name: "a strict schema", body: `{"model":"gpt-5.6-terra","input":"hi","text":{"format":{"type":"json_schema","name":"x","strict":true,"schema":` + strictSchema + `}}}`},
		{name: "a loose schema that is not strict", body: `{"model":"gpt-5.6-terra","input":"hi","text":{"format":{"type":"json_schema","name":"x","strict":false,"schema":{"type":"object"}}}}`},
		{name: "a loose tool that is not strict", body: `{"model":"gpt-5.6-terra","input":"hi","tools":[{"type":"function","name":"f","strict":false,"parameters":{"type":"object","properties":{"a":{"type":"string"}}}}]}`},
		{name: "a strict tool", body: `{"model":"gpt-5.6-terra","input":"hi","tools":[{"type":"function","name":"f","strict":true,"parameters":` + strictSchema + `}]}`},
		{
			name: "a call answered by its output",
			body: `{"model":"gpt-5.6-terra","input":[{"role":"user","content":"hi"},{"type":"function_call","call_id":"call_1","name":"f","arguments":"{}"},` +
				`{"type":"function_call_output","call_id":"call_1","output":"{}"}]}`,
		},
		{
			name: "a reasoning item with its encrypted content",
			body: `{"model":"gpt-5.6-terra","input":[{"type":"reasoning","id":"rs_1","summary":[],"encrypted_content":"gAAA"},{"role":"user","content":"hi"}]}`,
		},
		{
			name: "a namespace of deferred tools beside tool search",
			body: `{"model":"gpt-5.6-terra","input":"hi","tools":[{"type":"function","name":"g","strict":false,"parameters":{"type":"object"}},` +
				`{"type":"namespace","name":"pages","description":"The page map.",` +
				`"tools":[{"type":"function","name":"f","strict":false,"defer_loading":true,"parameters":{"type":"object"}}]},{"type":"tool_search"}]}`,
		},
		{
			name: "a top level deferred tool beside tool search",
			body: `{"model":"gpt-5.6-terra","input":"hi","tools":[{"type":"tool_search"},` +
				`{"type":"function","name":"f","strict":false,"defer_loading":true,"parameters":{"type":"object"}}]}`,
		},
		{
			name: "a namespace whose tools are all loaded at once",
			body: `{"model":"gpt-5.6-terra","input":"hi","tools":[{"type":"namespace","name":"pages","description":"The page map.",` +
				`"tools":[{"type":"function","name":"f","strict":false,"parameters":{"type":"object"}}]}]}`,
		},
		{
			name: "a replayed tool search before the namespaced call it loaded",
			body: `{"model":"gpt-5.6-terra","input":[{"role":"user","content":"hi"},` +
				`{"type":"tool_search_call","execution":"server","arguments":{"paths":["pages"]}},` +
				`{"type":"tool_search_output","execution":"server","tools":[]},` +
				`{"type":"function_call","call_id":"call_1","namespace":"pages","name":"f","arguments":"{}"},` +
				`{"type":"function_call_output","call_id":"call_1","output":"{}"}],"tools":[{"type":"tool_search"}]}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			server.Enqueue(openaitest.Text("ok").Reply())

			if got := post(t, server, "/responses", openaitest.DefaultKey, tc.body); got.Status != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", got.Status, got.Body)
			}
		})
	}
}
