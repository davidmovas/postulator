package openai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/openai"
	"github.com/davidmovas/postulator/internal/adapters/llm/openai/openaitest"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type heard struct {
	err    error
	done   *port.Delta
	texts  []string
	calls  []port.ToolCall
	after  int
	closed bool
}

func (h heard) text() string {
	return strings.Join(h.texts, "")
}

func listen(t *testing.T, deltas <-chan port.Delta) heard {
	t.Helper()

	var got heard
	deadline := time.After(20 * time.Second)
	for {
		select {
		case delta, open := <-deltas:
			if !open {
				got.closed = true
				return got
			}
			if got.done != nil || got.err != nil {
				got.after++
			}
			switch {
			case delta.Err != nil:
				got.err = delta.Err
			case delta.Done:
				final := delta
				got.done = &final
			case delta.Call != nil:
				got.calls = append(got.calls, *delta.Call)
			default:
				got.texts = append(got.texts, delta.Text)
			}
		case <-deadline:
			t.Fatal("the stream never closed")
		}
	}
}

func chat(text string) port.Request {
	req := write(text)
	req.Effort = llm.EffortNone
	return req
}

func TestAStreamDeliversTextThenOneFinalDelta(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Answer{
		Text:   "Koffein und Powder",
		Chunks: []string{"Koffein", " und", " Powder"},
		Usage:  openaitest.Usage{Input: 1200, Cached: 1024, CacheWrite: 64, Output: 40, Reasoning: 12},
		Tier:   "default",
	}.Stream())

	deltas, err := newClient(server).Stream(t.Context(), chat("hello"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	got := listen(t, deltas)

	if !reflect.DeepEqual(got.texts, []string{"Koffein", " und", " Powder"}) {
		t.Errorf("texts = %q, want the chunks in order", got.texts)
	}
	if got.err != nil || got.done == nil || got.after != 0 || !got.closed {
		t.Fatalf("stream = %+v, want one final delta, nothing after it and a closed channel", got)
	}
	want := llm.Usage{Input: 1200, CachedInput: 1024, CacheWrite: 64, Output: 40, Reasoning: 12, Total: 1240}
	if got.done.Usage == nil || *got.done.Usage != want {
		t.Errorf("usage = %+v, want %+v", got.done.Usage, want)
	}
	if got.done.Finish != port.FinishStop || got.done.Tier != llm.TierDefault {
		t.Errorf("final = %+v, want a stop on the default tier", got.done)
	}

	request := only(t, server)
	if request.Header.Get("Accept") != "text/event-stream" {
		t.Errorf("accept = %q, want an event stream", request.Header.Get("Accept"))
	}
}

func TestAStreamDeliversEachCallWhenItIsComplete(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Answer{
		Text: "Looking.",
		Calls: []openaitest.Call{
			{ID: "call_1", Name: "pages_list", Arguments: `{"limit":5}`},
			{ID: "call_2", Name: "pages_get", Arguments: `{"id":"p1"}`},
		},
	}.Stream())

	deltas, err := newClient(server).Stream(t.Context(), chat("list"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	got := listen(t, deltas)

	want := []port.ToolCall{
		{ID: "call_1", Name: "pages_list", Args: json.RawMessage(`{"limit":5}`)},
		{ID: "call_2", Name: "pages_get", Args: json.RawMessage(`{"id":"p1"}`)},
	}
	if !reflect.DeepEqual(got.calls, want) {
		t.Errorf("calls = %+v, want each call once, in order", got.calls)
	}
	if got.text() != "Looking." || got.done == nil {
		t.Errorf("stream = %+v, want the text and a final delta", got)
	}
}

func TestAStreamSaysWhyItStopped(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		answer openaitest.Answer
		finish port.FinishReason
		tier   llm.ServiceTier
		text   string
	}{
		{name: "cut at the ceiling", answer: openaitest.Truncated("Koff"), finish: port.FinishLength, tier: llm.TierDefault, text: "Koff"},
		{name: "refused", answer: openaitest.Refusal("no"), finish: port.FinishContentFilter, tier: llm.TierDefault},
		{name: "served on flex", answer: openaitest.Answer{Text: "slow", Tier: "flex"}, finish: port.FinishStop, tier: llm.TierFlex, text: "slow"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			server.Enqueue(tc.answer.Stream())

			deltas, err := newClient(server).Stream(t.Context(), chat("hello"))
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}
			got := listen(t, deltas)
			if got.done == nil || got.done.Finish != tc.finish || got.done.Tier != tc.tier || got.text() != tc.text {
				t.Errorf("stream = %+v (final %+v), want %s on %s with %q", got, got.done, tc.finish, tc.tier, tc.text)
			}
		})
	}
}

func TestTheChatRoundGoesOutWithItsToolsItsPairsAndItsCacheKey(t *testing.T) {
	t.Parallel()

	minimum := 1.0
	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("No pages yet.").Stream())

	deltas, err := newClient(server).Stream(t.Context(), port.Request{
		Ref:    ref("gpt-5.6-terra"),
		System: "You are the assistant.",
		Messages: []port.Message{
			{Role: port.RoleDeveloper, Text: "Site: Example."},
			{Role: port.RoleUser, Text: "List the pages."},
			{Role: port.RoleAssistant, Call: &port.ToolCall{ID: "call_1", Name: "pages_list", Args: json.RawMessage(`{"limit":5}`)}},
			{Role: port.RoleTool, Result: &port.ToolResult{CallID: "call_1", Output: json.RawMessage(`{"pages":[]}`)}},
		},
		Tools: []port.Tool{
			{Name: "pages_list", Description: "Lists pages.", Schema: &port.Schema{
				Type: port.SchemaObject, Properties: map[string]*port.Schema{"limit": {Type: port.SchemaInteger, Minimum: &minimum}},
			}},
			{Name: "pages_get", Description: "Reads a page.", Schema: &port.Schema{
				Type: port.SchemaObject, Properties: map[string]*port.Schema{"id": {Type: port.SchemaString}}, Required: []string{"id"},
			}},
		},
		MaxTokens: 2048,
		Effort:    llm.EffortNone,
		Tier:      llm.TierDefault,
		CacheKey:  "chat:conv-1",
		Meta:      port.CallMeta{ConversationID: "conv-1", Role: llm.RoleChat},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if got := listen(t, deltas); got.text() != "No pages yet." {
		t.Errorf("text = %q, want the answer", got.text())
	}
	assertGolden(t, "chat_request.json", only(t, server).Raw)
}

func TestAnInStreamRefusalBeforeAnyAnswerIsAFailedStart(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		reply openaitest.Reply
		want  errors.Code
		retry bool
	}{
		{name: "the probed credit refusal", reply: openaitest.StreamFailure(openaitest.Quota()), want: errors.NeedsHuman},
		{name: "a rate limit", reply: openaitest.StreamFailure(openaitest.RateLimit()), want: errors.RateLimited, retry: true},
		{name: "a server error", reply: openaitest.StreamFailure(openaitest.ServerError()), want: errors.External, retry: true},
		{name: "a refusal over http", reply: openaitest.QuotaExhausted(), want: errors.NeedsHuman},
		{name: "a server error over http", reply: openaitest.Failure(http.StatusServiceUnavailable, openaitest.ServerError()), want: errors.External, retry: true},
		{name: "a stream that ends before it answers", reply: openaitest.Stream(openaitest.Event{Name: "response.created", Data: `{"type":"response.created"}`}), want: errors.External, retry: true},
		{name: "an event that is not json", reply: openaitest.Stream(openaitest.Event{Name: "response.output_text.delta", Data: `{"type":`}), want: errors.External, retry: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			server.Enqueue(tc.reply)

			_, err := newClient(server).Stream(t.Context(), chat("hello"))
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("Stream = %v (%s), want %s", err, errors.CodeOf(err), tc.want)
			}
			if tc.retry != (kernelOf(t, err).Retry != nil) {
				t.Errorf("retry = %+v, want retryable %t", kernelOf(t, err).Retry, tc.retry)
			}
		})
	}
}

func TestAStreamWhoseBudgetRanOutIsRefusedWithTheReason(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Failure(http.StatusBadRequest, openaitest.Fault{
		Type:    "invalid_request_error",
		Message: "Could not finish the message because max_tokens or model output limit was reached.",
	}))

	_, err := newClient(server).Stream(t.Context(), chat("hello"))
	if !errors.IsCode(err, errors.Invalid) || kernelOf(t, err).Message != "the model used its whole output budget before it answered" {
		t.Fatalf("Stream = %v, want the exhausted budget named", err)
	}
}

func TestAStreamFailedWithoutAnErrorEventIsStillAFailure(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Stream(
		openaitest.Event{Name: "response.created", Data: `{"type":"response.created"}`},
		openaitest.Event{Name: "response.failed", Data: `{"type":"response.failed","response":{"status":"failed",` +
			`"error":{"code":"server_error","message":"The server had an error."}}}`},
	))

	_, err := newClient(server).Stream(t.Context(), chat("hello"))
	if !errors.IsCode(err, errors.External) || kernelOf(t, err).Details["code"] != "server_error" {
		t.Fatalf("Stream = %v, want the server error the failed response named", err)
	}
}

func TestTheProbedCreditRefusalInAStreamNamesItsCode(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.StreamFailure(openaitest.Quota()))

	_, err := newClient(server).Stream(t.Context(), chat("hello"))
	kernel := kernelOf(t, err)
	if kernel.Details["code"] != "credit_balance_exhausted" || kernel.Details["status"] != nil {
		t.Errorf("details = %v, want the provider's code and no invented status", kernel.Details)
	}
	if kernel.Message != quotaMessage {
		t.Errorf("message = %q, want the credit sentence", kernel.Message)
	}
}

func TestAFailureAfterTheModelSpokeEndsTheStreamWithIt(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Answer{Text: "half an answer", Chunks: []string{"half", " an answer"}}.StreamFailing(openaitest.ServerError()))

	deltas, err := newClient(server).Stream(t.Context(), chat("hello"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	got := listen(t, deltas)
	if got.text() != "half an answer" {
		t.Errorf("text = %q, want what came before the failure", got.text())
	}
	if !errors.IsCode(got.err, errors.External) || got.done != nil || got.after != 0 {
		t.Fatalf("stream = %+v, want one external failure as the last delta", got)
	}
}

func TestAStreamCutShortIsAFailure(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		events []openaitest.Event
	}{
		{name: "it ends without a final event", events: []openaitest.Event{
			{Name: "response.output_text.delta", Data: `{"type":"response.output_text.delta","delta":"half"}`},
		}},
		{name: "an event after the answer began is not json", events: []openaitest.Event{
			{Name: "response.output_text.delta", Data: `{"type":"response.output_text.delta","delta":"half"}`},
			{Name: "response.output_text.delta", Data: `{"type":`},
		}},
		{name: "a call arrives that cannot be called", events: []openaitest.Event{
			{Name: "response.output_text.delta", Data: `{"type":"response.output_text.delta","delta":"half"}`},
			{Name: "response.output_item.done", Data: `{"type":"response.output_item.done","item":{"type":"function_call","call_id":"c","name":"f","arguments":"{"}}`},
		}},
		{name: "the final event carries a call that cannot be called", events: []openaitest.Event{
			{Name: "response.output_text.delta", Data: `{"type":"response.output_text.delta","delta":"half"}`},
			{Name: "response.completed", Data: `{"type":"response.completed","response":{"status":"completed","output":[{"type":"function_call","name":"f"}]}}`},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			server.Enqueue(openaitest.Stream(tc.events...))

			deltas, err := newClient(server).Stream(t.Context(), chat("hello"))
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}
			got := listen(t, deltas)
			if got.text() != "half" || !errors.IsCode(got.err, errors.External) || got.done != nil {
				t.Errorf("stream = %+v, want the text and then an external failure", got)
			}
		})
	}
}

func TestAStreamToleratesWhatItDoesNotNeed(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Stream(
		openaitest.Event{Name: "response.created", Data: `{"type":"response.created","sequence_number":0}`},
		openaitest.Event{Name: "message", Data: `[DONE]`},
		openaitest.Event{Name: "response.reasoning_summary_text.delta", Data: `{"type":"response.reasoning_summary_text.delta","delta":{"odd":true}}`},
		openaitest.Event{Name: "response.output_text.delta", Data: `{"type":"response.output_text.delta","delta":"Koffein","obfuscation":"x9Yz"}`},
		openaitest.Event{Name: "message", Data: `[DONE]`},
		openaitest.Event{Name: "response.something_new", Data: `{"type":"response.something_new","payload":[1,2,3]}`},
		openaitest.Event{Name: "response.output_text.delta", Data: `{"type":"response.output_text.delta","delta":""}`},
		openaitest.Event{Name: "response.completed", Data: `{"type":"response.completed","response":{"status":"completed","service_tier":"default",` +
			`"output":[{"type":"message","content":[{"type":"output_text","text":"Koffein"}]},` +
			`{"type":"function_call","call_id":"call_9","name":"pages_list","arguments":""}],` +
			`"usage":{"input_tokens":5,"output_tokens":2}}}`},
		openaitest.Event{Name: "done", Data: `[DONE]`},
	))

	deltas, err := newClient(server).Stream(t.Context(), chat("hello"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	got := listen(t, deltas)
	if got.err != nil || got.text() != "Koffein" {
		t.Fatalf("stream = %+v, want the text without a failure", got)
	}
	if len(got.calls) != 1 || got.calls[0].ID != "call_9" || string(got.calls[0].Args) != "{}" {
		t.Errorf("calls = %+v, want the call the final event alone carried", got.calls)
	}
	if got.done == nil || got.done.Usage.Total != 7 {
		t.Errorf("final = %+v, want the usage", got.done)
	}
}

func TestAStreamWithoutDeltasStillDeliversTheAnswer(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Stream(
		openaitest.Event{Name: "response.incomplete", Data: `{"type":"response.incomplete","response":{"incomplete_details":{"reason":"max_output_tokens"},` +
			`"output":[{"type":"message","content":[{"type":"output_text","text":"whole"}]}]}}`},
	))

	deltas, err := newClient(server).Stream(t.Context(), chat("hello"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	got := listen(t, deltas)
	if got.text() != "whole" || got.done == nil || got.done.Finish != port.FinishLength {
		t.Errorf("stream = %+v (final %+v), want the text and a length stop", got, got.done)
	}
}

func TestAStreamCarriesAnEventOfSeveralMebibytes(t *testing.T) {
	t.Parallel()

	large := strings.Repeat("Koffein ", 4<<20/8)
	server := openaitest.New(t)
	server.Enqueue(openaitest.Answer{Text: large, Chunks: []string{large}}.Stream())

	deltas, err := newClient(server).Stream(t.Context(), chat("hello"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	got := listen(t, deltas)
	if got.err != nil || len(got.text()) != len(large) || got.done == nil {
		t.Fatalf("stream carried %d bytes with %v, want all %d", len(got.text()), got.err, len(large))
	}
}

func TestAStreamStopsWhenTheCallerLeaves(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	answer := openaitest.Answer{Text: "first second", Chunks: []string{"first", " second"}}.Stream()
	answer.Events[len(answer.Events)-1].Pause = 10 * time.Second
	server.Enqueue(answer)

	ctx, cancel := context.WithCancel(t.Context())
	deltas, err := newClient(server).Stream(ctx, chat("hello"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	first := <-deltas
	if first.Text != "first" {
		t.Fatalf("first delta = %+v, want the first chunk", first)
	}
	cancel()

	started := time.Now()
	got := listen(t, deltas)
	if waited := time.Since(started); waited > 5*time.Second {
		t.Errorf("the stream held on for %s after the caller left", waited)
	}
	if got.done != nil {
		t.Errorf("a cancelled stream still finished: %+v", got.done)
	}
}

func TestAStreamThatOutlivesTheClientsTimeoutFails(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	answer := openaitest.Answer{Text: "first second", Chunks: []string{"first", " second"}}.Stream()
	answer.Events[len(answer.Events)-1].Pause = 10 * time.Second
	server.Enqueue(answer)

	deltas, err := newClient(server, openai.WithTimeout(300*time.Millisecond)).Stream(t.Context(), chat("hello"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	got := listen(t, deltas)
	if !errors.IsCode(got.err, errors.External) || kernelOf(t, got.err).Retry == nil {
		t.Fatalf("stream = %+v, want a retryable external failure", got)
	}
}

func TestAStreamIsRefusedAtHomeLikeACompletion(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	client := openai.New(vault{}, models(), openai.WithBaseURL(server.URL()))

	if _, err := client.Stream(t.Context(), port.Request{Ref: ref("gpt-5.6-terra")}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("Stream without messages = %v, want %s", err, errors.Invalid)
	}
	if _, err := client.Stream(t.Context(), chat("hello")); !errors.IsCode(err, errors.Unauthorized) {
		t.Errorf("Stream without a key = %v, want %s", err, errors.Unauthorized)
	}
	if len(server.Requests()) != 0 {
		t.Error("a refused stream still left")
	}
}

func TestACancelledStreamStartIsCancelled(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("late").Stream().After(5 * time.Second))

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()
	if _, err := newClient(server).Stream(ctx, chat("hello")); !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("Stream = %v (%s), want %s", err, errors.CodeOf(err), errors.Cancelled)
	}
}
