package llm_test

import (
	"context"
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

func TestRequestValidate(t *testing.T) {
	t.Parallel()

	hot := 9.0
	cases := []struct {
		name    string
		mutate  func(*llm.Request)
		wantErr bool
	}{
		{name: "a plain request is valid"},
		{name: "the provider is required", mutate: func(r *llm.Request) { r.Ref.Provider = "" }, wantErr: true},
		{name: "the model is required", mutate: func(r *llm.Request) { r.Ref.Model = "" }, wantErr: true},
		{name: "messages are required", mutate: func(r *llm.Request) { r.Messages = nil }, wantErr: true},
		{name: "a role must be known", mutate: func(r *llm.Request) { r.Messages[0].Role = "system" }, wantErr: true},
		{name: "a message must not be empty", mutate: func(r *llm.Request) { r.Messages[0].Text = "" }, wantErr: true},
		{
			name:    "the last message comes from the user",
			mutate:  func(r *llm.Request) { r.Messages = append(r.Messages, llm.Message{Role: llm.RoleAssistant, Text: "hi"}) },
			wantErr: true,
		},
		{name: "the ceiling is not negative", mutate: func(r *llm.Request) { r.MaxTokens = -1 }, wantErr: true},
		{name: "the temperature is bounded", mutate: func(r *llm.Request) { r.Temperature = &hot }, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := llm.Request{Ref: ref(), Messages: userMessage("write"), MaxTokens: 128}
			if tc.mutate != nil {
				tc.mutate(&req)
			}

			err := req.Validate()
			if tc.wantErr != (err != nil) {
				t.Fatalf("Validate() = %v, want error %t", err, tc.wantErr)
			}
			if tc.wantErr && !errors.IsCode(err, errors.Invalid) {
				t.Errorf("Validate() code = %s, want %s", errors.CodeOf(err), errors.Invalid)
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
	Sections []section `json:"sections"`
	Score    float64   `json:"score"`
	Words    int       `json:"words"`
	Draft    bool      `json:"draft"`
	Note     *string   `json:"note,omitempty"`
	Internal string    `json:"-"`
	hidden   string
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
	if schema.Type != llm.SchemaObject || schema.Title != "draft" {
		t.Fatalf("schema = %+v, want an object titled draft", schema)
	}

	kinds := map[string]llm.SchemaType{
		"title":    llm.SchemaString,
		"tone":     llm.SchemaString,
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
		wantErr   bool
	}{
		{
			name:      "decodes the first answer",
			responses: []llm.Response{{Text: `{"heading":"Koffein","html":"<p>x</p>"}`, Usage: domain.Usage{Input: 10, Output: 4, Total: 14}}},
			wantTitle: "Koffein",
			wantUsage: domain.Usage{Input: 10, Output: 4, Total: 14},
			wantCalls: 1,
		},
		{
			name: "repairs once and sums the usage",
			responses: []llm.Response{
				{Text: "not json", Usage: domain.Usage{Input: 10, Output: 2, Total: 12}},
				{Text: `{"heading":"Powder","html":"<p>y</p>"}`, Usage: domain.Usage{Input: 20, Output: 4, Total: 24}},
			},
			wantTitle: "Powder",
			wantUsage: domain.Usage{Input: 30, Output: 6, Total: 36},
			wantCalls: 2,
		},
		{
			name: "gives up after the repair",
			responses: []llm.Response{
				{Text: "not json", Usage: domain.Usage{Input: 10, Output: 2, Total: 12}},
				{Text: "still not json", Usage: domain.Usage{Input: 10, Output: 2, Total: 12}},
			},
			wantUsage: domain.Usage{Input: 20, Output: 4, Total: 24},
			wantCalls: 2,
			wantErr:   true,
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

			if tc.wantErr {
				if !errors.IsCode(err, errors.Invalid) {
					t.Fatalf("Structured error = %v, want %s", err, errors.Invalid)
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
			if !strings.HasPrefix(first.System, "you write pages") || !strings.Contains(first.System, "single JSON object") {
				t.Errorf("system = %q, want the caller prompt plus the json instruction", first.System)
			}
			if tc.wantCalls > 1 {
				repaired := client.seen[1].Messages
				if len(repaired) != 2 || !strings.Contains(repaired[1].Text, "could not be decoded") {
					t.Errorf("repair messages = %+v, want the decoder error appended", repaired)
				}
				if len(first.Messages) != 1 {
					t.Errorf("first request messages = %+v, want the original single message", first.Messages)
				}
			}
		})
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
