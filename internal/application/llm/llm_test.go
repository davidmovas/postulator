package llm_test

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/llm"
	domain "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type scriptedClient struct {
	seen      []llm.Request
	responses []llm.Response
	err       error
}

func (c *scriptedClient) Complete(_ context.Context, req llm.Request) (llm.Response, error) {
	c.seen = append(c.seen, req)
	if c.err != nil {
		return llm.Response{}, c.err
	}
	resp := c.responses[0]
	if len(c.responses) > 1 {
		c.responses = c.responses[1:]
	}
	return resp, nil
}

func (c *scriptedClient) Stream(context.Context, llm.Request) (<-chan llm.Delta, error) {
	return nil, errors.New(errors.Internal, "the scripted client does not stream")
}

func ref() domain.ModelRef {
	return domain.ModelRef{Provider: "openai", Model: "gpt-5.1"}
}

func userMessage(text string) []llm.Message {
	return []llm.Message{{Role: llm.RoleUser, Text: text}}
}

func call(id, name, args string) llm.Message {
	return llm.Message{Role: llm.RoleAssistant, Call: &llm.ToolCall{ID: id, Name: name, Args: json.RawMessage(args)}}
}

func result(callID, output string) llm.Message {
	return llm.Message{Role: llm.RoleTool, Result: &llm.ToolResult{CallID: callID, Output: json.RawMessage(output)}}
}

func conversation(messages ...llm.Message) func(*llm.Request) {
	return func(r *llm.Request) { r.Messages = messages }
}

func searched(kind llm.SearchKind, payload string) llm.Message {
	return llm.Message{Role: llm.RoleAssistant, Search: &llm.ToolSearch{Kind: kind, Execution: "server", Payload: json.RawMessage(payload)}}
}

func namespaced(id, namespace, name string) llm.Message {
	return llm.Message{Role: llm.RoleAssistant, Call: &llm.ToolCall{ID: id, Name: name, Args: json.RawMessage(`{}`), Namespace: namespace}}
}

func deferred(name, group, description string) llm.Tool {
	return llm.Tool{
		Name: name, Description: "reads", Schema: &llm.Schema{Type: llm.SchemaObject},
		Deferred: &llm.ToolGroup{Name: group, Description: description},
	}
}

func TestRequestValidate(t *testing.T) {
	t.Parallel()

	hot := 9.0
	user := llm.Message{Role: llm.RoleUser, Text: "find the pages"}
	developer := llm.Message{Role: llm.RoleDeveloper, Text: "the site is example.com"}
	answer := llm.Message{Role: llm.RoleAssistant, Text: "here they are"}
	listing := llm.Tool{Name: "pages_list", Description: "lists pages", Schema: &llm.Schema{Type: llm.SchemaObject}}

	cases := []struct {
		name    string
		mutate  func(*llm.Request)
		wantErr string
	}{
		{name: "a plain request is valid"},
		{name: "the provider is required", mutate: func(r *llm.Request) { r.Ref.Provider = "" }, wantErr: "the model reference"},
		{name: "the model is required", mutate: func(r *llm.Request) { r.Ref.Model = "" }, wantErr: "the model reference"},
		{name: "messages are required", mutate: func(r *llm.Request) { r.Messages = nil }, wantErr: "at least one message"},
		{name: "a role must be known", mutate: func(r *llm.Request) { r.Messages[0].Role = "system" }, wantErr: "a message role"},
		{name: "a message must not be empty", mutate: func(r *llm.Request) { r.Messages[0].Text = "" }, wantErr: "must not be empty"},
		{
			name: "the last message comes from the user",
			mutate: func(r *llm.Request) {
				r.Messages = append(r.Messages, llm.Message{Role: llm.RoleAssistant, Text: "hi"})
			},
			wantErr: "the last message",
		},
		{name: "the ceiling is not negative", mutate: func(r *llm.Request) { r.MaxTokens = -1 }, wantErr: "the token ceiling"},
		{name: "the temperature is bounded", mutate: func(r *llm.Request) { r.Temperature = &hot }, wantErr: "the temperature"},
		{name: "developer context may precede the user", mutate: conversation(developer, user)},
		{name: "developer context may close the request", mutate: conversation(user, developer)},
		{
			name:   "a tool round ends on its result",
			mutate: conversation(user, call("call_1", "pages_list", `{"siteId":"s"}`), result("call_1", `{"pages":[]}`)),
		},
		{
			name: "two calls of one round are answered in turn",
			mutate: conversation(user, call("call_1", "pages_list", `{}`), call("call_2", "pages_get", `{"id":"p"}`),
				result("call_1", `{"pages":[]}`), result("call_2", `"gone"`)),
		},
		{
			name: "the user speaks again after a finished round",
			mutate: conversation(user, call("call_1", "pages_list", `{}`), result("call_1", `[]`), answer,
				llm.Message{Role: llm.RoleUser, Text: "thanks, now publish"}),
		},
		{
			name: "a message carries one thing",
			mutate: conversation(llm.Message{
				Role: llm.RoleAssistant, Text: "calling",
				Call: &llm.ToolCall{ID: "call_1", Name: "pages_list", Args: json.RawMessage(`{}`)},
			}, user),
			wantErr: "only one of",
		},
		{
			name: "a tool message carries its result and nothing else",
			mutate: conversation(user, call("call_1", "pages_list", `{}`), llm.Message{
				Role: llm.RoleTool, Text: "done",
				Result: &llm.ToolResult{CallID: "call_1", Output: json.RawMessage(`[]`)},
			}),
			wantErr: "only one of",
		},
		{
			name: "a tool call comes from the assistant",
			mutate: conversation(llm.Message{
				Role: llm.RoleUser, Call: &llm.ToolCall{ID: "call_1", Name: "pages_list", Args: json.RawMessage(`{}`)},
			}, user),
			wantErr: "a tool call must come from the assistant",
		},
		{
			name: "a tool result travels in a tool message",
			mutate: conversation(user, call("call_1", "pages_list", `{}`), llm.Message{
				Role: llm.RoleUser, Result: &llm.ToolResult{CallID: "call_1", Output: json.RawMessage(`[]`)},
			}),
			wantErr: "a tool result must travel in a tool message",
		},
		{
			name:    "a tool message carries a tool result",
			mutate:  conversation(user, call("call_1", "pages_list", `{}`), llm.Message{Role: llm.RoleTool, Text: "[]"}),
			wantErr: "a tool message must carry a tool result",
		},
		{
			name:    "a tool call has an identifier",
			mutate:  conversation(user, call("", "pages_list", `{}`), user),
			wantErr: "a tool call needs an identifier",
		},
		{
			name:    "a tool call names its tool",
			mutate:  conversation(user, call("call_1", "", `{}`), user),
			wantErr: "a tool call must name its tool",
		},
		{
			name:    "a tool call's arguments are JSON",
			mutate:  conversation(user, call("call_1", "pages_list", `{"siteId":`), user),
			wantErr: "arguments must be JSON",
		},
		{
			name:    "a tool call carries its arguments",
			mutate:  conversation(user, call("call_1", "pages_list", ``), user),
			wantErr: "arguments must be JSON",
		},
		{
			name:    "a tool result names the call it answers",
			mutate:  conversation(user, call("call_1", "pages_list", `{}`), result("", `[]`)),
			wantErr: "a tool result must name the call it answers",
		},
		{
			name:    "a tool result is JSON",
			mutate:  conversation(user, call("call_1", "pages_list", `{}`), result("call_1", `not json`)),
			wantErr: "a tool result must be JSON",
		},
		{
			name:    "a tool result answers a call",
			mutate:  conversation(user, result("call_9", `[]`)),
			wantErr: "a call made before it",
		},
		{
			name:    "a tool result follows its call",
			mutate:  conversation(user, result("call_1", `[]`), call("call_1", "pages_list", `{}`), user),
			wantErr: "a call made before it",
		},
		{
			name:    "a request does not end on a tool call",
			mutate:  conversation(user, call("call_1", "pages_list", `{}`)),
			wantErr: "the last message",
		},
		{name: "a known effort is accepted", mutate: func(r *llm.Request) { r.Effort = domain.EffortLow }},
		{name: "an effort must be known", mutate: func(r *llm.Request) { r.Effort = "extreme" }, wantErr: "the reasoning effort"},
		{name: "the flex tier is accepted", mutate: func(r *llm.Request) { r.Tier = domain.TierFlex }},
		{name: "a tier must be known", mutate: func(r *llm.Request) { r.Tier = "priority" }, wantErr: "the service tier"},
		{name: "a cache key is free text", mutate: func(r *llm.Request) { r.CacheKey = "agent:v1" }},
		{name: "tools may be offered", mutate: func(r *llm.Request) { r.Tools = []llm.Tool{listing, {Name: "pages_get"}} }},
		{
			name:    "a tool has a name",
			mutate:  func(r *llm.Request) { r.Tools = []llm.Tool{listing, {Description: "nameless"}} },
			wantErr: "a tool needs a name",
		},
		{
			name:    "tool names are unique",
			mutate:  func(r *llm.Request) { r.Tools = []llm.Tool{listing, listing} },
			wantErr: "two tools share one name",
		},
		{
			name: "a tool's parameters are an object",
			mutate: func(r *llm.Request) {
				r.Tools = []llm.Tool{{Name: "pages_list", Schema: &llm.Schema{Type: llm.SchemaString}}}
			},
			wantErr: "a tool's parameters must be an object",
		},
		{
			name: "deferred tools sit beside the tools sent whole",
			mutate: func(r *llm.Request) {
				r.Tools = []llm.Tool{listing, deferred("pages_get", "pages", "the page map"), deferred("pages_tree", "pages", "the page map")}
			},
		},
		{
			name:    "a deferred tool's group has a name",
			mutate:  func(r *llm.Request) { r.Tools = []llm.Tool{deferred("pages_get", "", "the page map")} },
			wantErr: "a group with a name and a description",
		},
		{
			name:    "a deferred tool's group says what it holds",
			mutate:  func(r *llm.Request) { r.Tools = []llm.Tool{deferred("pages_get", "pages", "")} },
			wantErr: "a group with a name and a description",
		},
		{
			name: "a group is described one way",
			mutate: func(r *llm.Request) {
				r.Tools = []llm.Tool{deferred("pages_get", "pages", "the page map"), deferred("pages_tree", "pages", "the tree")}
			},
			wantErr: "a tool group is described two ways",
		},
		{
			name: "a tool search is replayed before the namespaced call it loaded",
			mutate: conversation(user, searched(llm.SearchCall, `{"paths":["pages"]}`), searched(llm.SearchOutput, `[]`),
				namespaced("call_1", "pages", "pages_get"), result("call_1", `{}`)),
		},
		{
			name:    "a tool search comes from the assistant",
			mutate:  conversation(llm.Message{Role: llm.RoleUser, Search: &llm.ToolSearch{Kind: llm.SearchCall, Payload: json.RawMessage(`{}`)}}, user),
			wantErr: "a tool search must come from the assistant",
		},
		{
			name:    "a tool search is a call or its output",
			mutate:  conversation(user, searched("lookup", `{}`), user),
			wantErr: "a tool search is a call or its output",
		},
		{
			name:    "a tool search carries JSON",
			mutate:  conversation(user, searched(llm.SearchOutput, `[`), user),
			wantErr: "a tool search must carry JSON",
		},
		{
			name: "a tool search is a message of its own",
			mutate: conversation(llm.Message{
				Role: llm.RoleAssistant, Text: "searching",
				Search: &llm.ToolSearch{Kind: llm.SearchCall, Payload: json.RawMessage(`{}`)},
			}, user),
			wantErr: "only one of",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := llm.Request{Ref: ref(), Messages: userMessage("write"), MaxTokens: 128}
			if tc.mutate != nil {
				tc.mutate(&req)
			}

			err := req.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want no error", err)
				}
				return
			}
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Validate() = %v, want %s", err, errors.Invalid)
			}
			if _, message := errors.Describe(err); !strings.Contains(message, tc.wantErr) {
				t.Errorf("Validate() message = %q, want it to say %q", message, tc.wantErr)
			}
		})
	}
}

func TestTheRequestEncoding(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value any
		want  string
	}{
		{
			name: "a request that uses none of the new fields keeps its recorded fixtures findable",
			value: llm.Request{
				Ref: ref(), System: "you write", Messages: userMessage("write"), MaxTokens: 128,
				Meta: llm.CallMeta{RunID: "r"},
			},
			want: `{"ref":{"provider":"openai","model":"gpt-5.1"},"system":"you write",` +
				`"messages":[{"role":"user","text":"write"}],` +
				`"meta":{"runId":"r","itemId":"","step":"","conversationId":""},"maxTokens":128}`,
		},
		{
			name: "the tool fields, the effort, the tier and the cache key are camelCase",
			value: llm.Request{
				Ref: ref(),
				Messages: []llm.Message{
					call("call_1", "pages_list", `{"siteId":"s"}`),
					result("call_1", `{"pages":[]}`),
				},
				Meta:     llm.CallMeta{ConversationID: "c", Role: domain.RoleChat},
				Tools:    []llm.Tool{{Name: "pages_list", Description: "lists pages", Schema: &llm.Schema{Type: llm.SchemaObject}}},
				Effort:   domain.EffortLow,
				Tier:     domain.TierFlex,
				CacheKey: "agent",
			},
			want: `{"ref":{"provider":"openai","model":"gpt-5.1"},"system":"",` +
				`"messages":[{"role":"assistant","text":"","call":{"id":"call_1","name":"pages_list","args":{"siteId":"s"}}},` +
				`{"role":"tool","text":"","result":{"callId":"call_1","output":{"pages":[]}}}],` +
				`"meta":{"runId":"","itemId":"","step":"","conversationId":"c","role":"chat"},"maxTokens":0,` +
				`"tools":[{"name":"pages_list","description":"lists pages","schema":{"type":"object"}}],` +
				`"effort":"low","tier":"flex","cacheKey":"agent"}`,
		},
		{
			name:  "a plain response",
			value: llm.Response{Text: "hi", FinishReason: llm.FinishStop},
			want: `{"text":"hi","usage":{"input":0,"cachedInput":0,"cacheWrite":0,"output":0,"reasoning":0,"total":0},` +
				`"finishReason":"stop"}`,
		},
		{
			name: "a response that calls tools on flex",
			value: llm.Response{
				Usage:        domain.Usage{Input: 1, Output: 2, Reasoning: 1, Total: 3},
				FinishReason: llm.FinishStop,
				Calls:        []llm.ToolCall{{ID: "call_1", Name: "pages_list", Args: json.RawMessage(`{}`)}},
				Tier:         domain.TierFlex,
			},
			want: `{"text":"","usage":{"input":1,"cachedInput":0,"cacheWrite":0,"output":2,"reasoning":1,"total":3},` +
				`"finishReason":"stop",` +
				`"calls":[{"id":"call_1","name":"pages_list","args":{}}],"tier":"flex"}`,
		},
		{
			name: "a delta that carries a call and the end of the answer",
			value: llm.Delta{
				Done:   true,
				Call:   &llm.ToolCall{ID: "call_1", Name: "pages_list", Args: json.RawMessage(`{}`)},
				Finish: llm.FinishStop,
				Tier:   domain.TierDefault,
			},
			want: `{"text":"","done":true,"call":{"id":"call_1","name":"pages_list","args":{}},"finish":"stop","tier":"default"}`,
		},
		{
			name: "a deferred tool names its group and a replayed round carries the search and the namespace",
			value: llm.Request{
				Ref: ref(),
				Messages: []llm.Message{
					searched(llm.SearchCall, `{"paths":["pages"]}`),
					searched(llm.SearchOutput, `[]`),
					namespaced("call_1", "pages", "pages_get"),
				},
				Tools: []llm.Tool{deferred("pages_get", "pages", "the page map")},
			},
			want: `{"ref":{"provider":"openai","model":"gpt-5.1"},"system":"",` +
				`"messages":[{"role":"assistant","text":"","search":{"kind":"call","execution":"server","payload":{"paths":["pages"]}}},` +
				`{"role":"assistant","text":"","search":{"kind":"output","execution":"server","payload":[]}},` +
				`{"role":"assistant","text":"","call":{"id":"call_1","name":"pages_get","args":{},"namespace":"pages"}}],` +
				`"meta":{"runId":"","itemId":"","step":"","conversationId":""},"maxTokens":0,` +
				`"tools":[{"name":"pages_get","description":"reads","schema":{"type":"object"},` +
				`"deferred":{"name":"pages","description":"the page map"}}]}`,
		},
		{
			name: "a response and a delta carry the tool search the model ran",
			value: []any{
				llm.Response{Searches: []llm.ToolSearch{{Kind: llm.SearchOutput, CallID: "ts_1", Payload: json.RawMessage(`[]`)}}},
				llm.Delta{Search: &llm.ToolSearch{Kind: llm.SearchCall, Payload: json.RawMessage(`{}`)}},
			},
			want: `[{"text":"","usage":{"input":0,"cachedInput":0,"cacheWrite":0,"output":0,"reasoning":0,"total":0},` +
				`"finishReason":"","searches":[{"kind":"output","callId":"ts_1","payload":[]}]},` +
				`{"text":"","done":false,"search":{"kind":"call","payload":{}}}]`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if string(encoded) != tc.want {
				t.Fatalf("Marshal =\n%s\nwant\n%s", encoded, tc.want)
			}
		})
	}
}

type section struct {
	Heading string `json:"heading" description:"the section heading"`
	HTML    string `json:"html"`
}

type draft struct {
	Title    string    `json:"title" description:"the page title"`
	Tone     string    `json:"tone" enum:"formal,casual"`
	Tags     []string  `json:"tags,omitempty" enum:"seo,news"`
	Sections []section `json:"sections"`
	Score    float64   `json:"score" minimum:"0" maximum:"1"`
	Words    int       `json:"words"`
	Draft    bool      `json:"draft"`
	Note     *string   `json:"note,omitempty"`
	Internal string    `json:"-"`
}

type node struct {
	Name  string `json:"name"`
	Child *node  `json:"child,omitempty"`
}

type bag struct {
	Values map[string]string `json:"values"`
}

func TestSchemaFor(t *testing.T) {
	t.Parallel()

	schema, err := llm.SchemaFor[draft]()
	if err != nil {
		t.Fatalf("SchemaFor: %v", err)
	}
	if schema.Type != llm.SchemaObject {
		t.Fatalf("schema = %+v, want an object", schema)
	}

	kinds := map[string]llm.SchemaType{
		"title":    llm.SchemaString,
		"tone":     llm.SchemaString,
		"tags":     llm.SchemaArray,
		"sections": llm.SchemaArray,
		"score":    llm.SchemaNumber,
		"words":    llm.SchemaInteger,
		"draft":    llm.SchemaBoolean,
		"note":     llm.SchemaString,
	}
	if len(schema.Properties) != len(kinds) {
		t.Fatalf("properties = %v, want %d entries", schema.Properties, len(kinds))
	}
	for name, want := range kinds {
		property, ok := schema.Properties[name]
		if !ok {
			t.Fatalf("property %q is missing", name)
		}
		if property.Type != want {
			t.Errorf("property %q type = %s, want %s", name, property.Type, want)
		}
	}

	if got := schema.Properties["title"].Description; got != "the page title" {
		t.Errorf("title description = %q, want the tag value", got)
	}
	if got := strings.Join(schema.Properties["tone"].Enum, ","); got != "formal,casual" {
		t.Errorf("tone enum = %q, want formal,casual", got)
	}
	if got := strings.Join(schema.Required, ","); got != "draft,score,sections,title,tone,words" {
		t.Errorf("required = %q, want the non-optional fields sorted", got)
	}

	items := schema.Properties["sections"].Items
	if items == nil || items.Type != llm.SchemaObject {
		t.Fatalf("section items = %+v, want an object", items)
	}
	if got := items.Properties["heading"].Description; got != "the section heading" {
		t.Errorf("heading description = %q, want the tag value", got)
	}

	tags := schema.Properties["tags"]
	if len(tags.Enum) != 0 {
		t.Errorf("tags enum = %v, want the choice on the items rather than the list", tags.Enum)
	}
	if tags.Items == nil || strings.Join(tags.Items.Enum, ",") != "seo,news" {
		t.Errorf("tag items = %+v, want the enum seo,news", tags.Items)
	}

	score := schema.Properties["score"]
	if score.Minimum == nil || *score.Minimum != 0 {
		t.Errorf("score minimum = %v, want 0", score.Minimum)
	}
	if score.Maximum == nil || *score.Maximum != 1 {
		t.Errorf("score maximum = %v, want 1", score.Maximum)
	}
}

func TestNoGoTypeNameReachesTheModel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		make func() (*llm.Schema, error)
	}{
		{name: "the object itself", make: llm.SchemaFor[draft]},
		{name: "a nested object", make: llm.SchemaFor[section]},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			schema, err := tc.make()
			if err != nil {
				t.Fatalf("SchemaFor: %v", err)
			}
			if named := keywords(schema); len(named) != 0 {
				t.Errorf("the schema carries %v, which names a Go type the model has no use for", named)
			}
		})
	}
}

func keywords(schema *llm.Schema) []string {
	if schema == nil {
		return nil
	}

	out := make([]string, 0)
	encoded, err := json.Marshal(schema)
	if err != nil {
		return []string{err.Error()}
	}
	var fields map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(encoded, &fields); unmarshalErr != nil {
		return []string{unmarshalErr.Error()}
	}
	if held, ok := fields["title"]; ok {
		out = append(out, "a title of "+string(held))
	}
	out = append(out, keywords(schema.Items)...)
	for _, property := range schema.Properties {
		out = append(out, keywords(property)...)
	}
	return out
}

func TestSchemaForRefusesUnsupportedTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		make func() (*llm.Schema, error)
	}{
		{name: "a recursive struct", make: llm.SchemaFor[node]},
		{name: "a map field", make: llm.SchemaFor[bag]},
		{name: "a channel", make: llm.SchemaFor[chan int]},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := tc.make(); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("SchemaFor error = %v, want %s", err, errors.Invalid)
			}
		})
	}
}

func TestStructured(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		responses []llm.Response
		wantTitle string
		wantUsage domain.Usage
		wantCalls int
		wantCode  errors.Code
		wantWhy   string
	}{
		{
			name:      "decodes the first answer",
			responses: []llm.Response{{Text: `{"heading":"Koffein","html":"<p>x</p>"}`, Usage: domain.Usage{Input: 10, Output: 4, Total: 14}}},
			wantTitle: "Koffein",
			wantUsage: domain.Usage{Input: 10, Output: 4, Total: 14},
			wantCalls: 1,
		},
		{
			name: "hands a malformed answer to the step's retry instead of asking again",
			responses: []llm.Response{
				{Text: "not json", Usage: domain.Usage{Input: 10, Output: 2, Total: 12}},
				{Text: `{"heading":"Powder","html":"<p>y</p>"}`, Usage: domain.Usage{Input: 20, Output: 4, Total: 24}},
			},
			wantUsage: domain.Usage{Input: 10, Output: 2, Total: 12},
			wantCalls: 1,
			wantCode:  errors.External,
			wantWhy:   llm.ReasonMalformedAnswer,
		},
		{
			name: "refuses an answer of another shape",
			responses: []llm.Response{
				{Text: `{"heading":7}`, Usage: domain.Usage{Input: 10, Output: 2, Total: 12}},
			},
			wantUsage: domain.Usage{Input: 10, Output: 2, Total: 12},
			wantCalls: 1,
			wantCode:  errors.External,
			wantWhy:   llm.ReasonMalformedAnswer,
		},
		{
			name: "stops at an answer that stopped for length",
			responses: []llm.Response{
				{Text: `{"heading":"Koff`, Usage: domain.Usage{Input: 10, Output: 8, Total: 18}, FinishReason: llm.FinishLength},
			},
			wantUsage: domain.Usage{Input: 10, Output: 8, Total: 18},
			wantCalls: 1,
			wantCode:  errors.External,
			wantWhy:   llm.ReasonOutputTruncated,
		},
		{
			name: "holds a filtered answer for a human",
			responses: []llm.Response{
				{Text: "", Usage: domain.Usage{Input: 10, Output: 0, Total: 10}, FinishReason: llm.FinishContentFilter},
			},
			wantUsage: domain.Usage{Input: 10, Output: 0, Total: 10},
			wantCalls: 1,
			wantCode:  errors.NeedsHuman,
			wantWhy:   llm.ReasonContentFilter,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := &scriptedClient{responses: tc.responses}
			got, usage, err := llm.Structured[section](t.Context(), client, llm.Request{
				Ref:      ref(),
				System:   "you write pages",
				Messages: userMessage("write a section"),
			})

			if tc.wantCode != "" {
				if !errors.IsCode(err, tc.wantCode) {
					t.Fatalf("Structured error = %v, want %s", err, tc.wantCode)
				}
				if why := reasonOf(err); why != tc.wantWhy {
					t.Fatalf("Structured error reason = %v, want %s", why, tc.wantWhy)
				}
			} else if err != nil {
				t.Fatalf("Structured: %v", err)
			}
			if got.Heading != tc.wantTitle {
				t.Errorf("heading = %q, want %q", got.Heading, tc.wantTitle)
			}
			if usage != tc.wantUsage {
				t.Errorf("usage = %+v, want %+v", usage, tc.wantUsage)
			}
			if len(client.seen) != tc.wantCalls {
				t.Fatalf("calls = %d, want %d", len(client.seen), tc.wantCalls)
			}

			first := client.seen[0]
			if first.Schema == nil || first.Schema.Type != llm.SchemaObject {
				t.Errorf("schema = %+v, want the derived object schema", first.Schema)
			}
			if first.System != "you write pages" {
				t.Errorf("system = %q, want the caller's prompt as it was written; the strict schema says the shape", first.System)
			}
			if len(first.Messages) != 1 {
				t.Errorf("request messages = %+v, want the caller's single message", first.Messages)
			}
		})
	}
}

func TestAMalformedAnswerNamesItsModelAndWhyItFailed(t *testing.T) {
	t.Parallel()

	client := &scriptedClient{responses: []llm.Response{{Text: "Sure! Here is the section."}}}
	_, _, err := llm.Structured[section](t.Context(), client, llm.Request{Ref: ref(), Messages: userMessage("go")})

	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatalf("Structured error = %v, want a kernel error", err)
	}
	if kernel.Details["model"] != ref().String() || kernel.Details["reason"] != llm.ReasonMalformedAnswer {
		t.Fatalf("the refusal carries %v", kernel.Details)
	}
	if !strings.Contains(err.Error(), "invalid character") {
		t.Fatalf("the refusal does not keep the decoder's reason: %v", err)
	}
}

func TestStructuredPropagatesTheClientError(t *testing.T) {
	t.Parallel()

	client := &scriptedClient{err: errors.New(errors.RateLimited, "slow down")}
	if _, _, err := llm.Structured[section](t.Context(), client, llm.Request{Ref: ref(), Messages: userMessage("go")}); !errors.IsCode(err, errors.RateLimited) {
		t.Fatalf("Structured error = %v, want %s", err, errors.RateLimited)
	}
}

func TestStructuredRejectsAnUnsupportedType(t *testing.T) {
	t.Parallel()

	client := &scriptedClient{responses: []llm.Response{{Text: "{}"}}}
	if _, _, err := llm.Structured[node](t.Context(), client, llm.Request{Ref: ref(), Messages: userMessage("go")}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Structured error = %v, want %s", err, errors.Invalid)
	}
	if len(client.seen) != 0 {
		t.Errorf("calls = %d, want the schema to fail before the request", len(client.seen))
	}
}

func reasonOf(err error) string {
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		return ""
	}
	reason, ok := kernel.Details["reason"].(string)
	if !ok {
		return ""
	}
	return reason
}
