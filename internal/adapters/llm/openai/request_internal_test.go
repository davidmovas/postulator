package openai

import (
	"encoding/json"
	"strings"
	"testing"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func terra() llm.ModelInfo {
	return llm.ModelInfo{
		Ref:             llm.ModelRef{Provider: "openai", Model: "gpt-5.6-terra"},
		ReasoningEffort: llm.EffortMedium,
		ContextTokens:   1_050_000,
		MaxOutputTokens: 128_000,
		InputUSDPerM:    2, CachedInputUSDPerM: 0.2, CacheWriteUSDPerM: 2.5, OutputUSDPerM: 12,
		FlexInputUSDPerM: 1, FlexCachedInputUSDPerM: 0.1, FlexCacheWriteUSDPerM: 1.25, FlexOutputUSDPerM: 6,
		Reasoning: true,
	}
}

func plainModel() llm.ModelInfo {
	return llm.ModelInfo{
		Ref:             llm.ModelRef{Provider: "openai", Model: "gpt-4.1"},
		ContextTokens:   1_000_000,
		MaxOutputTokens: 32_768,
		InputUSDPerM:    2, OutputUSDPerM: 8,
	}
}

func asked(mutate func(*port.Request)) port.Request {
	req := port.Request{
		Ref:      llm.ModelRef{Provider: "openai", Model: "gpt-5.6-terra"},
		Messages: []port.Message{{Role: port.RoleUser, Text: "write"}},
	}
	if mutate != nil {
		mutate(&req)
	}
	return req
}

func TestTheReasoningEffortIsAlwaysSentToAModelThatReasons(t *testing.T) {
	t.Parallel()

	minimalRow := terra()
	minimalRow.ReasoningEffort = effortMinimal
	silentRow := terra()
	silentRow.ReasoningEffort = ""

	cases := []struct {
		name  string
		req   port.Request
		model catalogRow
		want  string
	}{
		{name: "the effort the caller asked for", req: asked(func(r *port.Request) { r.Effort = llm.EffortLow }), model: listed(terra()), want: "low"},
		{name: "the catalog's effort when none is asked", req: asked(nil), model: listed(terra()), want: "medium"},
		{name: "none is sent, never left to the server's medium", req: asked(func(r *port.Request) { r.Effort = llm.EffortNone }), model: listed(terra()), want: "none"},
		{name: "a catalog row that says minimal is sent low", req: asked(nil), model: listed(minimalRow), want: "low"},
		{name: "a row without an effort is sent medium", req: asked(nil), model: listed(silentRow), want: "medium"},
		{name: "a model that does not reason is sent no effort", req: asked(func(r *port.Request) { r.Effort = llm.EffortHigh }), model: listed(plainModel())},
		{name: "an unlisted model keeps the asked effort", req: asked(func(r *port.Request) { r.Effort = llm.EffortHigh }), model: catalogRow{}, want: "high"},
		{name: "an unlisted model asked for nothing is sent nothing", req: asked(nil), model: catalogRow{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body, err := requestOf(tc.req, tc.model, false)
			if err != nil {
				t.Fatalf("requestOf: %v", err)
			}
			switch {
			case tc.want == "" && body.Reasoning != nil:
				t.Errorf("reasoning = %+v, want none sent", body.Reasoning)
			case tc.want != "" && (body.Reasoning == nil || body.Reasoning.Effort != tc.want):
				t.Errorf("reasoning = %+v, want effort %s", body.Reasoning, tc.want)
			}
		})
	}
}

func TestTheOutputCeilingMakesRoomForReasoning(t *testing.T) {
	t.Parallel()

	tight := terra()
	tight.MaxOutputTokens = 9000

	cases := []struct {
		name  string
		req   port.Request
		model catalogRow
		want  int
	}{
		{name: "no ceiling asked is no ceiling sent", req: asked(nil), model: listed(terra())},
		{name: "the asked words plus the medium allowance", req: asked(func(r *port.Request) { r.MaxTokens = 6000 }), model: listed(terra()), want: 6000 + 8192},
		{name: "the asked effort's allowance", req: asked(func(r *port.Request) { r.MaxTokens = 600; r.Effort = llm.EffortLow }), model: listed(terra()), want: 600 + 2048},
		{name: "effort none adds nothing", req: asked(func(r *port.Request) { r.MaxTokens = 600; r.Effort = llm.EffortNone }), model: listed(terra()), want: 600},
		{name: "clamped to what the model can write", req: asked(func(r *port.Request) { r.MaxTokens = 6000 }), model: listed(tight), want: 9000},
		{name: "never under the provider's floor", req: asked(func(r *port.Request) { r.MaxTokens = 4 }), model: listed(plainModel()), want: 16},
		{name: "a model that does not reason gets what was asked", req: asked(func(r *port.Request) { r.MaxTokens = 256 }), model: listed(plainModel()), want: 256},
		{name: "an unlisted model gets what was asked", req: asked(func(r *port.Request) { r.MaxTokens = 256 }), model: catalogRow{}, want: 256},
		{name: "an unlisted model asked to reason gets the allowance", req: asked(func(r *port.Request) { r.MaxTokens = 256; r.Effort = llm.EffortLow }), model: catalogRow{}, want: 256 + 2048},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body, err := requestOf(tc.req, tc.model, false)
			if err != nil {
				t.Fatalf("requestOf: %v", err)
			}
			if body.MaxOutput != tc.want {
				t.Errorf("max_output_tokens = %d, want %d", body.MaxOutput, tc.want)
			}
		})
	}
}

func TestFlexIsSentOnlyWhereTheCatalogPricesIt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		tier  llm.ServiceTier
		model catalogRow
		want  string
	}{
		{name: "flex on a flex row", tier: llm.TierFlex, model: listed(terra()), want: "flex"},
		{name: "flex on a row without flex prices", tier: llm.TierFlex, model: listed(plainModel()), want: "default"},
		{name: "flex on an unlisted model", tier: llm.TierFlex, model: catalogRow{}, want: "default"},
		{name: "the default asked", tier: llm.TierDefault, model: listed(terra()), want: "default"},
		{name: "nothing asked", model: listed(terra()), want: "default"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body, err := requestOf(asked(func(r *port.Request) { r.Tier = tc.tier }), tc.model, false)
			if err != nil {
				t.Fatalf("requestOf: %v", err)
			}
			if body.ServiceTier != tc.want {
				t.Errorf("service_tier = %s, want %s", body.ServiceTier, tc.want)
			}
		})
	}
}

func TestTheCacheIsLeftOutOfASingleShot(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("writer:site-0f7c", 5)
	tools := []port.Tool{{Name: "pages_list"}}

	cases := []struct {
		name     string
		key      string
		tools    []port.Tool
		wantKey  string
		explicit bool
	}{
		{name: "no tools and no key opt out of the implicit write", explicit: true},
		{name: "a key keeps the implicit cache", key: "writer:site-1", wantKey: "writer:site-1"},
		{name: "tools keep the implicit cache", tools: tools},
		{name: "a key of sixty four characters is sent as it is", key: strings.Repeat("k", 64), wantKey: strings.Repeat("k", 64)},
		{name: "a longer key is sent as its hash", key: long, wantKey: hashed(long)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body, err := requestOf(asked(func(r *port.Request) { r.CacheKey = tc.key; r.Tools = tc.tools }), listed(terra()), false)
			if err != nil {
				t.Fatalf("requestOf: %v", err)
			}
			if body.CacheKey != tc.wantKey {
				t.Errorf("prompt_cache_key = %q, want %q", body.CacheKey, tc.wantKey)
			}
			if len(body.CacheKey) > 64 {
				t.Errorf("prompt_cache_key has %d characters, the provider refuses more than 64", len(body.CacheKey))
			}
			if tc.explicit != (body.CacheOptions != nil && body.CacheOptions.Mode == "explicit") {
				t.Errorf("prompt_cache_options = %+v, want explicit %t", body.CacheOptions, tc.explicit)
			}
		})
	}

	first, again, other := hashed(long), hashed(long), hashed(long+"x")
	if first != again || first == other {
		t.Error("the hashed key is not a stable function of the key")
	}
}

func hashed(key string) string {
	body, err := requestOf(asked(func(r *port.Request) { r.CacheKey = key }), catalogRow{}, false)
	if err != nil {
		return ""
	}
	return body.CacheKey
}

func TestARequestCarriesWhatTheCallerAskedAndNothingTheProviderRefuses(t *testing.T) {
	t.Parallel()

	warm := 0.7
	req := asked(func(r *port.Request) {
		r.System = "You write pages."
		r.Temperature = &warm
		r.Tools = []port.Tool{{Name: "pages_list", Description: "Lists pages."}, {Name: "pages_get"}}
	})

	body, err := requestOf(req, listed(terra()), true)
	if err != nil {
		t.Fatalf("requestOf: %v", err)
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	sent := string(encoded)

	for _, want := range []string{`"store":false`, `"instructions":"You write pages."`, `"stream":true`,
		`"stream_options":{"include_obfuscation":false}`, `"name":"pages_list"`, `"strict":false`} {
		if !strings.Contains(sent, want) {
			t.Errorf("the request lacks %s: %s", want, sent)
		}
	}
	for _, unwanted := range []string{"temperature", "top_p", `"text"`, `"prompt_cache_options"`} {
		if strings.Contains(sent, unwanted) {
			t.Errorf("the request carries %s: %s", unwanted, sent)
		}
	}
	if strings.Index(sent, `"pages_list"`) > strings.Index(sent, `"pages_get"`) {
		t.Errorf("the tools left the caller's order: %s", sent)
	}
}

func TestARequestWithoutStreamingSaysNothingOfIt(t *testing.T) {
	t.Parallel()

	body, err := requestOf(asked(nil), listed(terra()), false)
	if err != nil {
		t.Fatalf("requestOf: %v", err)
	}
	if body.Stream || body.StreamOptions != nil || body.Instructions != "" || body.Tools != nil || body.Text != nil {
		t.Errorf("body = %+v, want only what was asked", body)
	}
}

func TestARequestWithAnArraySchemaIsRefused(t *testing.T) {
	t.Parallel()

	_, err := requestOf(asked(func(r *port.Request) { r.Schema = &port.Schema{Type: port.SchemaArray} }), listed(terra()), false)
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("requestOf = %v, want %s", err, errors.Invalid)
	}
}

func TestItemsKeepTheirTurnAndTheirPairs(t *testing.T) {
	t.Parallel()

	messages := []port.Message{
		{Role: port.RoleDeveloper, Text: "Site: Example."},
		{Role: port.RoleUser, Text: "List the pages."},
		{Role: port.RoleAssistant, Text: "Looking."},
		{Role: port.RoleAssistant, Call: &port.ToolCall{ID: "call_1", Name: "pages_list", Args: json.RawMessage(`{"limit": 5}`)}},
		{Role: port.RoleTool, Result: &port.ToolResult{CallID: "call_1", Output: json.RawMessage(`{"pages":[]}`)}},
	}

	const want = `[
	  {"role": "developer", "content": "Site: Example."},
	  {"role": "user", "content": "List the pages."},
	  {"role": "assistant", "content": "Looking."},
	  {"type": "function_call", "call_id": "call_1", "name": "pages_list", "arguments": "{\"limit\": 5}"},
	  {"type": "function_call_output", "call_id": "call_1", "output": "{\"pages\":[]}"}
	]`
	if got := canonical(t, itemsOf(messages)); got != canonicalText(t, want) {
		t.Errorf("items =\n%s\nwant\n%s", got, canonicalText(t, want))
	}

	encoded, err := json.Marshal(itemsOf(messages[3:4]))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if string(encoded) != `[{"type":"function_call","call_id":"call_1","name":"pages_list","arguments":"{\"limit\": 5}"}]` {
		t.Errorf("a call goes out as %s, want its fields in order and no id", encoded)
	}
}
