package fake_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var offered = []port.Tool{{Name: "pages_tree"}, {Name: "sites_list"}, {Name: "pages_update"}}

func asking(messages ...port.Message) port.Request {
	return port.Request{
		Ref:      llm.ModelRef{Provider: "openai", Model: "gpt-5.6-terra"},
		System:   "you are a fake",
		Messages: messages,
		Tools:    offered,
	}
}

func user(text string) port.Message {
	return port.Message{Role: port.RoleUser, Text: text}
}

type exchange struct {
	calls []port.ToolCall
	text  string
}

func converse(t *testing.T, client *fake.Client, history []port.Message, prompt string,
	answer func(port.ToolCall) string) exchange {
	t.Helper()

	messages := append(append([]port.Message(nil), history...), user(prompt))
	var got exchange
	for range 8 {
		resp, err := client.Complete(t.Context(), asking(messages...))
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if len(resp.Calls) == 0 {
			got.text = resp.Text
			return got
		}
		for _, call := range resp.Calls {
			got.calls = append(got.calls, call)
			messages = append(messages,
				port.Message{Role: port.RoleAssistant, Call: &call},
				port.Message{Role: port.RoleTool, Result: &port.ToolResult{CallID: call.ID, Output: json.RawMessage(answer(call))}},
			)
		}
	}
	t.Fatal("the scripted conversation never answered")
	return got
}

func succeeded(port.ToolCall) string {
	return `{"ok":true}`
}

func names(calls []port.ToolCall) string {
	named := make([]string, 0, len(calls))
	for _, call := range calls {
		named = append(named, call.Name)
	}
	return strings.Join(named, ",")
}

func TestAConversationWalksItsToolDirectives(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		prompt    string
		answer    func(port.ToolCall) string
		wantCalls string
		wantText  string
	}{
		{
			name:      "two calls and a final answer",
			prompt:    `TOOL:pages_tree{"a":1}` + "\n" + `TOOL:sites_list{}` + "\nFAKE: all done",
			answer:    succeeded,
			wantCalls: "pages_tree,sites_list",
			wantText:  "all done",
		},
		{
			name:     "no directive answers plainly",
			prompt:   "hello",
			answer:   succeeded,
			wantText: "ok",
		},
		{
			name:      "a call with no final answer says what it ran",
			prompt:    "TOOL:pages_tree{}",
			answer:    succeeded,
			wantCalls: "pages_tree",
			wantText:  "ran pages_tree",
		},
		{
			name:      "a call that failed is named as failed",
			prompt:    "TOOL:sites_list{}",
			answer:    func(port.ToolCall) string { return `{"error":"the site is gone"}` },
			wantCalls: "sites_list",
			wantText:  "ran sites_list (error)",
		},
		{
			name:      "the same tool called twice is two calls",
			prompt:    "TOOL:pages_tree{}\nTOOL:pages_tree{}\nFAKE: twice",
			answer:    succeeded,
			wantCalls: "pages_tree,pages_tree",
			wantText:  "twice",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := converse(t, fake.New(), nil, tc.prompt, tc.answer)
			if names(got.calls) != tc.wantCalls {
				t.Fatalf("calls = %q, want %q", names(got.calls), tc.wantCalls)
			}
			if got.text != tc.wantText {
				t.Fatalf("text = %q, want %q", got.text, tc.wantText)
			}

			seen := map[string]bool{}
			for _, call := range got.calls {
				if call.ID == "" || seen[call.ID] {
					t.Fatalf("the call ids are %+v, want each one set and distinct", got.calls)
				}
				seen[call.ID] = true
				if !json.Valid(call.Args) {
					t.Fatalf("the arguments %s are not JSON", call.Args)
				}
			}
		})
	}
}

func TestAScriptedCallCarriesItsArguments(t *testing.T) {
	t.Parallel()

	got := converse(t, fake.New(), nil, `TOOL:pages_update{"id":"p-1","metaTitle":"Espresso"}`+"\nFAKE: done", succeeded)
	if len(got.calls) != 1 {
		t.Fatalf("calls = %+v", got.calls)
	}

	var args map[string]string
	if err := json.Unmarshal(got.calls[0].Args, &args); err != nil || args["id"] != "p-1" || args["metaTitle"] != "Espresso" {
		t.Fatalf("arguments = %s, %v", got.calls[0].Args, err)
	}
}

func TestOnlyTheCurrentTurnCountsItsCalls(t *testing.T) {
	t.Parallel()

	client := fake.New()
	earlier := []port.Message{
		{Role: port.RoleDeveloper, Text: "the site holds three pages"},
		user("TOOL:pages_tree{}\nFAKE: one tree"),
		{Role: port.RoleAssistant, Call: &port.ToolCall{ID: "fake-call-1-1", Name: "pages_tree", Args: json.RawMessage(`{}`)}},
		{Role: port.RoleTool, Result: &port.ToolResult{CallID: "fake-call-1-1", Output: json.RawMessage(`{"ok":true}`)}},
		{Role: port.RoleAssistant, Text: "one tree"},
		{Role: port.RoleDeveloper, Text: "the site holds three pages"},
	}

	got := converse(t, client, earlier, "TOOL:sites_list{}\nFAKE: one listing", succeeded)
	if names(got.calls) != "sites_list" || got.text != "one listing" {
		t.Fatalf("the second turn ran %q and said %q", names(got.calls), got.text)
	}
	if got.calls[0].ID == "fake-call-1-1" {
		t.Fatalf("the second turn reused the first turn's call id %q", got.calls[0].ID)
	}
}

func TestAConversationFailsOnDemand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		prompt string
		want   errors.Code
	}{
		{name: "ERROR names a code", prompt: "ERROR:RATE_LIMITED", want: errors.RateLimited},
		{name: "FAIL names a code", prompt: "FAIL:UNAUTHORIZED", want: errors.Unauthorized},
		{name: "FAIL under a sentence", prompt: "show me the error row\nFAIL:EXTERNAL", want: errors.External},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := fake.New()
			if _, err := client.Complete(t.Context(), asking(user(tc.prompt))); !errors.IsCode(err, tc.want) {
				t.Fatalf("Complete = %v, want %s", err, tc.want)
			}
			if _, err := client.Stream(t.Context(), asking(user(tc.prompt))); !errors.IsCode(err, tc.want) {
				t.Fatalf("Stream = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestAConversationAnswersFromAScript(t *testing.T) {
	t.Parallel()

	script := func(prompt string) fake.Turn {
		if strings.Contains(prompt, "rename") {
			return fake.Turn{
				Tool: "pages_update",
				Args: json.RawMessage(`{"id":"p-1","metaTitle":"Best espresso machines under $500"}`),
				Text: "I retitled the page.",
			}
		}
		if strings.Contains(prompt, "list") {
			return fake.Turn{Tool: "sites_list", Text: "Two sites."}
		}
		return fake.Turn{Text: "Your graph covers eleven espresso topics."}
	}

	cases := []struct {
		name      string
		prompt    string
		wantCalls string
		wantText  string
	}{
		{name: "prose only", prompt: "how does my graph look?", wantText: "Your graph covers eleven espresso topics."},
		{name: "one write call", prompt: "please rename that page", wantCalls: "pages_update", wantText: "I retitled the page."},
		{name: "a call without arguments", prompt: "list the sites", wantCalls: "sites_list", wantText: "Two sites."},
		{name: "a directive outranks the script", prompt: "FAKE: the directive answered", wantText: "the directive answered"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := converse(t, fake.New(fake.WithScript(script)), nil, tc.prompt, succeeded)
			if names(got.calls) != tc.wantCalls || got.text != tc.wantText {
				t.Fatalf("the turn ran %q and said %q, want %q and %q", names(got.calls), got.text, tc.wantCalls, tc.wantText)
			}
			for _, call := range got.calls {
				if !json.Valid(call.Args) {
					t.Fatalf("the arguments %q are not JSON", call.Args)
				}
			}
		})
	}
}

func TestAConversationStreamsItsCallsAndThenItsAnswer(t *testing.T) {
	t.Parallel()

	client := fake.New()
	opening := user("TOOL:pages_tree{}\nFAKE: the tree is empty")

	first := drain(t, client, asking(opening))
	if len(first.calls) != 1 || first.text != "" || !first.done || first.usage.Output == 0 {
		t.Fatalf("the first round streamed %+v", first)
	}

	call := first.calls[0]
	second := drain(t, client, asking(opening,
		port.Message{Role: port.RoleAssistant, Call: &call},
		port.Message{Role: port.RoleTool, Result: &port.ToolResult{CallID: call.ID, Output: json.RawMessage(`{"roots":[]}`)}},
	))
	if len(second.calls) != 0 || second.text != "the tree is empty" || !second.done {
		t.Fatalf("the second round streamed %+v", second)
	}
	if second.usage.Input <= first.usage.Input {
		t.Fatalf("the second round read %d tokens and the first %d; the longer prompt must count more",
			second.usage.Input, first.usage.Input)
	}
	if second.finish != port.FinishStop || second.tier != llm.TierDefault {
		t.Fatalf("the second round ended with %q on %q", second.finish, second.tier)
	}
}

type streamed struct {
	calls  []port.ToolCall
	text   string
	usage  llm.Usage
	finish port.FinishReason
	tier   llm.ServiceTier
	done   bool
}

func drain(t *testing.T, client *fake.Client, req port.Request) streamed {
	t.Helper()

	deltas, err := client.Stream(t.Context(), req)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	var got streamed
	for delta := range deltas {
		got.text += delta.Text
		if delta.Call != nil {
			got.calls = append(got.calls, *delta.Call)
		}
		if delta.Done {
			got.done, got.finish, got.tier = true, delta.Finish, delta.Tier
			if delta.Usage != nil {
				got.usage = *delta.Usage
			}
		}
	}
	return got
}

func TestAFlexRequestIsServedOnFlex(t *testing.T) {
	t.Parallel()

	req := asking(user("FAKE: cheap"))
	req.Tier = llm.TierFlex

	resp, err := fake.New().Complete(t.Context(), req)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Tier != llm.TierFlex {
		t.Fatalf("tier = %q, want flex", resp.Tier)
	}
}
