package openaitest_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/openai/openaitest"
)

func served(t *testing.T, reply openaitest.Reply) answer {
	t.Helper()

	server := openaitest.New(t)
	server.Enqueue(reply)
	return post(t, server, "/responses", openaitest.DefaultKey, minimalBody)
}

func field(t *testing.T, value any, path ...any) any {
	t.Helper()

	current := value
	for _, step := range path {
		switch key := step.(type) {
		case string:
			object, ok := current.(map[string]any)
			if !ok {
				t.Fatalf("at %v the value %v is not an object", step, current)
			}
			current = object[key]
		case int:
			list, ok := current.([]any)
			if !ok || key >= len(list) {
				t.Fatalf("at %v the value %v is not a long enough list", step, current)
			}
			current = list[key]
		}
	}
	return current
}

func TestAnAnswerIsServedAsAResponsesObject(t *testing.T) {
	t.Parallel()

	reply := openaitest.Answer{
		Text:  "Koffein und Powder",
		Calls: []openaitest.Call{{ID: "call_1", Name: "pages_list", Arguments: `{"limit":5}`}},
		Tier:  "flex",
		Usage: openaitest.Usage{Input: 1200, Cached: 1024, CacheWrite: 100, Output: 300, Reasoning: 200},
	}.Reply()

	got := served(t, reply)
	if got.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200", got.Status)
	}
	body := decode(t, got.Body)

	checks := []struct {
		name string
		path []any
		want any
	}{
		{name: "the object", path: []any{"object"}, want: "response"},
		{name: "the status", path: []any{"status"}, want: "completed"},
		{name: "the model", path: []any{"model"}, want: openaitest.DefaultModel},
		{name: "the served tier", path: []any{"service_tier"}, want: "flex"},
		{name: "the reasoning item comes first", path: []any{"output", 0, "type"}, want: "reasoning"},
		{name: "the message", path: []any{"output", 1, "content", 0, "text"}, want: "Koffein und Powder"},
		{name: "the part type", path: []any{"output", 1, "content", 0, "type"}, want: "output_text"},
		{name: "the call id", path: []any{"output", 2, "call_id"}, want: "call_1"},
		{name: "the call arguments", path: []any{"output", 2, "arguments"}, want: `{"limit":5}`},
		{name: "the input tokens", path: []any{"usage", "input_tokens"}, want: float64(1200)},
		{name: "the cached tokens", path: []any{"usage", "input_tokens_details", "cached_tokens"}, want: float64(1024)},
		{name: "the cache writes", path: []any{"usage", "input_tokens_details", "cache_write_tokens"}, want: float64(100)},
		{name: "the output tokens", path: []any{"usage", "output_tokens"}, want: float64(300)},
		{name: "the reasoning tokens", path: []any{"usage", "output_tokens_details", "reasoning_tokens"}, want: float64(200)},
		{name: "the total", path: []any{"usage", "total_tokens"}, want: float64(1500)},
	}
	for _, check := range checks {
		if got := field(t, body, check.path...); !reflect.DeepEqual(got, check.want) {
			t.Errorf("%s = %v, want %v", check.name, got, check.want)
		}
	}
}

func TestAnAnswerTellsWhyItStopped(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		answer openaitest.Answer
		status string
		reason any
		part   string
	}{
		{name: "a finished answer", answer: openaitest.Text("done"), status: "completed", reason: nil, part: "output_text"},
		{name: "a truncated answer", answer: openaitest.Truncated(`{"title":"Koff`), status: "incomplete", reason: "max_output_tokens", part: "output_text"},
		{name: "a refusal", answer: openaitest.Refusal("I can't help with that."), status: "completed", reason: nil, part: "refusal"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body := decode(t, served(t, tc.answer.Reply()).Body)
			if got := field(t, body, "status"); got != tc.status {
				t.Errorf("status = %v, want %s", got, tc.status)
			}
			details := field(t, body, "incomplete_details")
			if tc.reason == nil && details != nil {
				t.Errorf("incomplete details = %v, want none", details)
			}
			if tc.reason != nil && field(t, details, "reason") != tc.reason {
				t.Errorf("incomplete details = %v, want %v", details, tc.reason)
			}
			if got := field(t, body, "output", 0, "content", 0, "type"); got != tc.part {
				t.Errorf("part = %v, want %s", got, tc.part)
			}
		})
	}
}

func TestAStructuredAnswerCarriesTheValueAsJSONText(t *testing.T) {
	t.Parallel()

	answer := openaitest.Structured(t, map[string]any{"title": "Koffein", "note": nil})
	if answer.Text != `{"note":null,"title":"Koffein"}` {
		t.Fatalf("text = %s, want the encoded value", answer.Text)
	}

	tb := &recorder{}
	if broken := openaitest.Structured(tb, func() {}); broken.Text != "" || tb.failures() != 1 {
		t.Errorf("an unencodable value gave %q and %d failures, want none and one", broken.Text, tb.failures())
	}
}

func TestCallsAreServedOneItemEach(t *testing.T) {
	t.Parallel()

	body := decode(t, served(t, openaitest.Calls(
		openaitest.Call{ID: "call_1", Name: "pages_list", Arguments: `{}`},
		openaitest.Call{ID: "call_2", Name: "pages_get", Arguments: `{"id":"p1"}`},
	).Reply()).Body)

	output, ok := field(t, body, "output").([]any)
	if !ok || len(output) != 2 {
		t.Fatalf("output = %v, want two calls", field(t, body, "output"))
	}
	for i, want := range []string{"pages_list", "pages_get"} {
		if field(t, output, i, "type") != "function_call" || field(t, output, i, "name") != want {
			t.Errorf("output[%d] = %v, want a call to %s", i, output[i], want)
		}
	}
}

func eventTypes(t *testing.T, read []frame) []string {
	t.Helper()

	types := make([]string, 0, len(read))
	for i, each := range read {
		var payload struct {
			Type     string `json:"type"`
			Sequence *int   `json:"sequence_number"`
		}
		if err := json.Unmarshal([]byte(each.data), &payload); err != nil {
			t.Fatalf("decode frame %d: %v", i, err)
		}
		if payload.Type != each.name {
			t.Errorf("frame %d is named %s but typed %s", i, each.name, payload.Type)
		}
		if payload.Sequence == nil || *payload.Sequence != i {
			t.Errorf("frame %d carries sequence %v, want %d", i, payload.Sequence, i)
		}
		types = append(types, payload.Type)
	}
	return types
}

func TestAStreamedAnswerFollowsTheRealEventOrder(t *testing.T) {
	t.Parallel()

	answer := openaitest.Answer{
		Text:   "Koffein und Powder",
		Chunks: []string{"Koffein", " und Powder"},
		Calls:  []openaitest.Call{{ID: "call_1", Name: "pages_list", Arguments: `{}`}},
		Usage:  openaitest.Usage{Input: 12, Output: 4, Reasoning: 2},
	}

	got := served(t, answer.Stream())
	if !strings.HasPrefix(got.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("content type = %q, want an event stream", got.Header.Get("Content-Type"))
	}
	read := frames(t, got.Body)

	want := []string{
		"response.created", "response.in_progress",
		"response.output_item.added", "response.output_item.done",
		"response.output_item.added", "response.content_part.added",
		"response.output_text.delta", "response.output_text.delta",
		"response.output_text.done", "response.content_part.done", "response.output_item.done",
		"response.output_item.added", "response.function_call_arguments.delta",
		"response.function_call_arguments.done", "response.output_item.done",
		"response.completed",
	}
	if types := eventTypes(t, read); !reflect.DeepEqual(types, want) {
		t.Fatalf("events = %v\nwant     %v", types, want)
	}

	deltas := []string{}
	for _, each := range read {
		if each.name == "response.output_text.delta" {
			deltas = append(deltas, text(t, field(t, decode(t, each.data), "delta")))
		}
	}
	if !reflect.DeepEqual(deltas, answer.Chunks) {
		t.Errorf("deltas = %v, want the chunks", deltas)
	}

	final := decode(t, read[len(read)-1].data)
	if field(t, final, "response", "usage", "output_tokens") != float64(4) {
		t.Errorf("the final event carries %v, want the usage", field(t, final, "response", "usage"))
	}
	if field(t, final, "response", "output", 1, "content", 0, "text") != "Koffein und Powder" {
		t.Errorf("the final event carries %v, want the whole answer", field(t, final, "response", "output"))
	}
}

func TestAStreamedAnswerWithoutChunksSendsItsTextAtOnce(t *testing.T) {
	t.Parallel()

	read := frames(t, served(t, openaitest.Text("whole").Stream()).Body)

	var deltas []string
	for _, each := range read {
		if each.name == "response.output_text.delta" {
			deltas = append(deltas, text(t, field(t, decode(t, each.data), "delta")))
		}
	}
	if !reflect.DeepEqual(deltas, []string{"whole"}) {
		t.Errorf("deltas = %v, want the text in one delta", deltas)
	}
}

func TestATruncatedStreamEndsIncomplete(t *testing.T) {
	t.Parallel()

	read := frames(t, served(t, openaitest.Truncated("Koff").Stream()).Body)
	final := read[len(read)-1]
	if final.name != "response.incomplete" {
		t.Fatalf("the stream ends with %s, want response.incomplete", final.name)
	}
	if got := field(t, decode(t, final.data), "response", "incomplete_details", "reason"); got != "max_output_tokens" {
		t.Errorf("reason = %v, want max_output_tokens", got)
	}
}

func TestARefusedStreamSaysSoInItsMessage(t *testing.T) {
	t.Parallel()

	read := frames(t, served(t, openaitest.Refusal("no").Stream()).Body)
	types := eventTypes(t, read)
	if !strings.Contains(strings.Join(types, " "), "response.refusal.delta") {
		t.Errorf("events = %v, want the refusal streamed", types)
	}
	final := decode(t, read[len(read)-1].data)
	if field(t, final, "response", "output", 0, "content", 0, "refusal") != "no" {
		t.Errorf("final = %v, want the refusal part", final)
	}
}

func TestAFailureCarriesTheErrorEnvelope(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		reply  openaitest.Reply
		status int
		kind   string
		code   any
		param  any
	}{
		{name: "the probe's quota refusal", reply: openaitest.QuotaExhausted(), status: http.StatusTooManyRequests, kind: "insufficient_quota", code: "credit_balance_exhausted"},
		{name: "a flex capacity refusal", reply: openaitest.FlexCapacity(), status: http.StatusTooManyRequests, kind: "invalid_request_error", code: "resource_unavailable"},
		{name: "a server error", reply: openaitest.Failure(http.StatusInternalServerError, openaitest.ServerError()), status: http.StatusInternalServerError, kind: "server_error"},
		{
			name:   "a refused parameter",
			reply:  openaitest.Failure(http.StatusBadRequest, openaitest.Fault{Type: "invalid_request_error", Code: "unsupported_value", Param: "reasoning.effort", Message: "no"}),
			status: http.StatusBadRequest, kind: "invalid_request_error", code: "unsupported_value", param: "reasoning.effort",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := served(t, tc.reply)
			if got.Status != tc.status {
				t.Fatalf("status = %d, want %d", got.Status, tc.status)
			}
			fault := errorOf(t, got.Body)
			if fault["type"] != tc.kind || fault["code"] != tc.code || fault["param"] != tc.param {
				t.Errorf("fault = %v, want type %s code %v param %v", fault, tc.kind, tc.code, tc.param)
			}
			if text(t, fault["message"]) == "" {
				t.Errorf("fault = %v, want a message", fault)
			}
			if _, present := fault["param"]; !present {
				t.Errorf("fault = %v, want param present even when null", fault)
			}
		})
	}
}

func TestTheQuotaRefusalIsTheProbedBody(t *testing.T) {
	t.Parallel()

	got := served(t, openaitest.QuotaExhausted())
	if got.Header.Get("Retry-After") != "" {
		t.Errorf("the quota refusal names a delay %q, the real one names none", got.Header.Get("Retry-After"))
	}
	fault := errorOf(t, got.Body)
	if !strings.HasPrefix(text(t, fault["message"]), "You have no credits remaining.") {
		t.Errorf("message = %v, want the probed sentence", fault["message"])
	}
}

func TestAStreamFailureFollowsTheProbedSequence(t *testing.T) {
	t.Parallel()

	got := served(t, openaitest.StreamFailure(openaitest.Quota()))
	if got.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200 as the probe saw", got.Status)
	}
	read := frames(t, got.Body)
	if types := eventTypes(t, read); !reflect.DeepEqual(types, []string{"response.created", "response.in_progress", "error", "response.failed"}) {
		t.Fatalf("events = %v, want the probed sequence", types)
	}

	failure := decode(t, read[2].data)
	if field(t, failure, "error", "code") != "credit_balance_exhausted" || field(t, failure, "error", "type") != "insufficient_quota" {
		t.Errorf("error event = %v, want the nested quota error", failure)
	}
	failed := decode(t, read[3].data)
	if field(t, failed, "response", "status") != "failed" || field(t, failed, "response", "error", "code") != "credit_balance_exhausted" {
		t.Errorf("failed event = %v, want the failed response", failed)
	}
}

func TestAStreamCanFailAfterItHasSpoken(t *testing.T) {
	t.Parallel()

	read := frames(t, served(t, openaitest.Text("half").StreamFailing(openaitest.ServerError())).Body)
	types := eventTypes(t, read)
	if types[len(types)-2] != "error" || types[len(types)-1] != "response.failed" {
		t.Fatalf("events = %v, want an error and a failed response at the end", types)
	}
	if !strings.Contains(strings.Join(types, " "), "response.output_text.delta") {
		t.Errorf("events = %v, want text before the failure", types)
	}
}
