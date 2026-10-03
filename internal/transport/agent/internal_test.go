package agent

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strings"
	"testing"
	"time"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func said(role llm.Role, text string) llm.Message {
	return llm.Message{Role: role, Text: text}
}

func called(id, name, args string) llm.Message {
	return llm.Message{Role: llm.RoleAssistant, Call: &llm.ToolCall{ID: id, Name: name, Args: json.RawMessage(args)}}
}

func answered(id, output string) llm.Message {
	return llm.Message{Role: llm.RoleTool, Result: &llm.ToolResult{CallID: id, Output: json.RawMessage(output)}}
}

func TestTrimKeepsTheNewestWholeTurns(t *testing.T) {
	t.Parallel()

	items := []llm.Message{
		said(llm.RoleDeveloper, "the site holds three pages"),
		said(llm.RoleUser, strings.Repeat("a", 400)),
		said(llm.RoleAssistant, strings.Repeat("b", 400)),
		said(llm.RoleDeveloper, "the site holds four pages"),
		said(llm.RoleUser, "the newest question"),
		called("call-1", "pages_tree", `{}`),
		answered("call-1", `{"ok":true}`),
		said(llm.RoleAssistant, "the newest answer"),
	}

	cases := []struct {
		name   string
		budget int
		want   int
	}{
		{name: "a history inside its budget is kept whole", budget: 100000, want: len(items)},
		{name: "no budget keeps everything", budget: 0, want: len(items)},
		{name: "an older turn over the budget is dropped whole", budget: 600, want: 5},
		{name: "the newest turn is kept even over the budget", budget: 10, want: 5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			kept := trim(items, tc.budget)
			if len(kept) != tc.want {
				t.Fatalf("trim kept %d messages, want %d", len(kept), tc.want)
			}
			if tc.want < len(items) && kept[0].Role != llm.RoleDeveloper {
				t.Fatalf("the trimmed history starts with %+v, want a turn's developer message", kept[0])
			}
		})
	}

	if trim(nil, 10) != nil {
		t.Error("trimming nothing answers nothing")
	}
	if kept := trim([]llm.Message{said(llm.RoleAssistant, strings.Repeat("a", 500))}, 100); len(kept) != 0 {
		t.Fatalf("a history over its budget that starts no turn is kept as %+v", kept)
	}
}

func TestTrimNeverSeparatesACallFromItsResult(t *testing.T) {
	t.Parallel()

	items := []llm.Message{
		said(llm.RoleDeveloper, "context one"),
		said(llm.RoleUser, "first"),
		called("call-1", "pages_tree", `{}`),
		answered("call-1", `{"roots":["`+strings.Repeat("x", 300)+`"]}`),
		said(llm.RoleAssistant, "one tree"),
		said(llm.RoleDeveloper, "context two"),
		said(llm.RoleUser, "second"),
		called("call-2", "pages_tree", `{}`),
		answered("call-2", `{"roots":[]}`),
	}

	for budget := 1; budget < 1200; budget += 7 {
		kept := trim(items, budget)
		calls, results := map[string]bool{}, map[string]bool{}
		for _, item := range kept {
			if item.Call != nil {
				calls[item.Call.ID] = true
			}
			if item.Result != nil {
				results[item.Result.CallID] = true
			}
		}
		if fmt.Sprint(calls) != fmt.Sprint(results) {
			t.Fatalf("a budget of %d kept the calls %v and the results %v", budget, calls, results)
		}
	}
}

type heldHistory struct {
	body    []byte
	version int
	err     error
}

func (h *heldHistory) Load(context.Context, string) (body []byte, version int, err error) {
	if h.err != nil {
		return nil, 0, h.err
	}
	if h.body == nil {
		return nil, 0, errors.New(errors.NotFound, "nothing stored yet")
	}
	return h.body, h.version, nil
}

func (h *heldHistory) Save(_ context.Context, _ string, body []byte, version int, _ time.Time) error {
	h.body = body
	h.version = version
	return nil
}

const fenceBytes = 64

func fatResult(t *testing.T, rows int) string {
	t.Helper()

	items := make([]any, 0, rows)
	for i := range rows {
		items = append(items, map[string]any{
			"id":    fmt.Sprintf("page-%03d", i),
			"path":  fmt.Sprintf("/coffee/espresso/%03d/", i),
			"title": strings.Repeat("Espresso ", 8),
		})
	}
	encoded, err := json.Marshal(agentapp.Fence(map[string]any{"items": items, "hasMore": true, "nextCursor": "c-42"}))
	if err != nil {
		t.Fatalf("encode the fixture: %v", err)
	}
	return string(encoded)
}

func remembering(store historyStore) memory {
	return memory{
		store: store, clock: clock.NewFake(time.Date(2026, 9, 22, 9, 0, 0, 0, time.UTC)), conversationID: "c1",
		budget: agentapp.DefaultHistoryBudgetChars, cap: agentapp.DefaultHistoryToolResultBytes,
	}
}

func TestAStoredToolResultReplaysShortenedAndStillDecodes(t *testing.T) {
	t.Parallel()

	fat := fatResult(t, 128)
	if len(fat) < agentapp.DefaultMaxToolResultBytes {
		t.Fatalf("the fixture is %d bytes, want at least the in-turn ceiling", len(fat))
	}

	store := &heldHistory{}
	held := remembering(store)
	written := []llm.Message{
		said(llm.RoleDeveloper, "the site holds 128 pages"),
		said(llm.RoleUser, "list the espresso pages"),
		called("call-1", "pages_list", `{}`),
		answered("call-1", fat),
		said(llm.RoleAssistant, "a hundred and twenty pages"),
	}
	if err := held.save(t.Context(), written); err != nil {
		t.Fatalf("save: %v", err)
	}
	if store.version != historyVersion || !strings.Contains(string(store.body), `"format":"responses/1"`) {
		t.Fatalf("the stored body is %s at version %d", store.body, store.version)
	}

	replayed, err := held.load(t.Context())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(replayed) != len(written) || replayed[1].Text != "list the espresso pages" {
		t.Fatalf("the replayed history is %+v", replayed)
	}

	shortened := replayed[3].Result
	if shortened == nil || shortened.CallID != "call-1" {
		t.Fatalf("the shortened result lost its call: %+v", replayed[3])
	}
	if len(shortened.Output) >= len(fat) || len(shortened.Output) > agentapp.DefaultHistoryToolResultBytes+fenceBytes {
		t.Fatalf("the stored result is %d bytes of the original %d", len(shortened.Output), len(fat))
	}

	var document map[string]any
	if err = json.Unmarshal(shortened.Output, &document); err != nil {
		t.Fatalf("the shortened result no longer decodes: %v", err)
	}
	if document[agentapp.UntrustedMarker] != true {
		t.Fatalf("the shortened result lost its fence: %+v", document)
	}
	inner, ok := document[agentapp.UntrustedData].(map[string]any)
	if !ok || inner[agentapp.TruncatedKey] != true {
		t.Fatalf("the shortened result does not say it was shortened: %+v", document)
	}
	kept, ok := inner[agentapp.ResultKey].(map[string]any)
	if !ok || kept["nextCursor"] != "c-42" || kept["hasMore"] != true {
		t.Fatalf("the shortened result lost the fields that say how to ask for the rest: %+v", inner)
	}
	if rows, listed := kept["items"].([]any); !listed || len(rows) == 0 || len(rows) >= 128 {
		t.Fatalf("the shortened result kept %v of 128 rows", kept["items"])
	}
}

func TestAResultThatFitsIsStoredUntouched(t *testing.T) {
	t.Parallel()

	items := []llm.Message{said(llm.RoleUser, "hello"), answered("call-1", `{"ok":true}`), answered("call-2", `[1,2]`)}

	kept := shorten(items, agentapp.DefaultHistoryToolResultBytes)
	if string(kept[1].Result.Output) != `{"ok":true}` || string(kept[2].Result.Output) != `[1,2]` {
		t.Fatalf("a result inside the ceiling was rewritten: %+v", kept)
	}
	if len(shorten(nil, 100)) != 0 {
		t.Error("shortening nothing answers nothing")
	}
	if got := shorten(items, 0); &got[0] != &items[0] {
		t.Error("no ceiling shortens nothing")
	}
}

func TestAnErrorResultIsShortenedWithoutAFence(t *testing.T) {
	t.Parallel()

	long := `{"error":"` + strings.Repeat("the site refused the write ", 200) + `"}`
	kept := shorten([]llm.Message{answered("call-1", long)}, 600)

	var document map[string]any
	if err := json.Unmarshal(kept[0].Result.Output, &document); err != nil {
		t.Fatalf("the shortened error no longer decodes: %v", err)
	}
	if document[agentapp.TruncatedKey] != true || document[agentapp.UntrustedMarker] != nil {
		t.Fatalf("the shortened error reads %+v", document)
	}
}

func TestAnyOtherHistoryLoadsEmpty(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
	}{
		{name: "gollem's history", body: `{"version":3,"llType":"openai","messages":[{"role":"user","contents":[]}]}`},
		{name: "a newer format", body: `{"format":"responses/2","items":[{"role":"user","text":"hi"}]}`},
		{name: "a body that is not JSON", body: `{"format":`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			loaded, err := remembering(&heldHistory{body: []byte(tc.body), version: 3}).load(t.Context())
			if err != nil || len(loaded) != 0 {
				t.Fatalf("load = %+v, %v; want an empty history", loaded, err)
			}
		})
	}
}

func TestAMemoryWithoutAConversationKeepsNothing(t *testing.T) {
	t.Parallel()

	store := &heldHistory{}
	held := remembering(store)
	held.conversationID = ""
	if err := held.save(t.Context(), []llm.Message{said(llm.RoleUser, "hello")}); err != nil || store.body != nil {
		t.Fatalf("save = %v, stored %s", err, store.body)
	}

	failing := remembering(&heldHistory{err: errors.New(errors.Internal, "the disk is gone")})
	if _, err := failing.load(t.Context()); !errors.IsCode(err, errors.Internal) {
		t.Fatalf("load of a failing store = %v", err)
	}
}

func TestTheHistoryCeilingIsTheSettingTheTurnCarries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		spec agentapp.RunSpec
		want int
	}{
		{
			name: "the turn carries what the setting says",
			spec: agentapp.RunSpec{HistoryToolResult: 8192, MaxToolResult: 16384},
			want: 8192,
		},
		{
			name: "a turn carrying none takes the shipped default",
			spec: agentapp.RunSpec{MaxToolResult: 262144},
			want: agentapp.DefaultHistoryToolResultBytes,
		},
		{
			name: "what the model may read in the turn no longer decides it",
			spec: agentapp.RunSpec{HistoryToolResult: 512, MaxToolResult: 262144},
			want: 512,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := historyCeiling(tc.spec); got != tc.want {
				t.Fatalf("historyCeiling = %d, want %d", got, tc.want)
			}
		})
	}

	if resultCeiling(agentapp.RunSpec{}) != agentapp.DefaultMaxToolResultBytes || resultCeiling(agentapp.RunSpec{MaxToolResult: 99}) != 99 {
		t.Fatal("the result ceiling does not follow the turn")
	}
}

func TestObjectOfWrapsWhatIsNotAnObject(t *testing.T) {
	t.Parallel()

	object, err := objectOf(map[string]any{"id": "p1", "wpId": int64(9007199254740993)})
	if err != nil || object["id"] != "p1" {
		t.Fatalf("objectOf = %v, %v", object, err)
	}
	if encoded := string(encode(object)); !strings.Contains(encoded, "9007199254740993") {
		t.Fatalf("a wide id lost its digits: %s", encoded)
	}

	wrapped, err := objectOf([]string{"a", "b"})
	if err != nil {
		t.Fatalf("objectOf of a list: %v", err)
	}
	items, ok := wrapped[resultKey].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("objectOf of a list = %v", wrapped)
	}

	if _, err = objectOf(make(chan int)); !errors.IsCode(err, errors.Internal) {
		t.Fatalf("objectOf of something unencodable = %v", err)
	}
}

func TestConvertNamesTheFailure(t *testing.T) {
	t.Parallel()

	stoppedCtx, stop := context.WithCancel(t.Context())
	stop()

	cases := []struct {
		name string
		ctx  context.Context
		err  error
		want errors.Code
	}{
		{name: "a coded error passes through", ctx: t.Context(), err: errors.New(errors.RateLimited, "slow down"), want: errors.RateLimited},
		{name: "a cancelled context is a stopped turn", ctx: t.Context(), err: context.Canceled, want: errors.Cancelled},
		{name: "a failure while the turn is stopping is a stopped turn", ctx: stoppedCtx, err: errors.New(errors.External, "gone"), want: errors.Cancelled},
		{name: "an unknown failure is the provider's", ctx: t.Context(), err: stderrors.New("broken pipe"), want: errors.External},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := convert(tc.ctx, tc.err); !errors.IsCode(got, tc.want) {
				t.Fatalf("convert = %v, want %s", got, tc.want)
			}
		})
	}
	if convert(t.Context(), nil) != nil {
		t.Error("convert of nothing is nothing")
	}
	if got := unfinished(stoppedCtx); !errors.IsCode(got, errors.Cancelled) {
		t.Errorf("a stream cut by a stopped turn = %v", got)
	}
	if got := unfinished(t.Context()); !errors.IsCode(got, errors.External) {
		t.Errorf("a stream that ended early = %v", got)
	}
}

func TestEncodeAnswersAnObjectForNothing(t *testing.T) {
	t.Parallel()

	if got := string(encode(nil)); got != "{}" {
		t.Fatalf("encode(nil) = %s", got)
	}
	if got := string(encode(map[string]any{"a": 1})); got != `{"a":1}` {
		t.Fatalf("encode = %s", got)
	}
	if got := string(encode(map[string]any{"a": make(chan int)})); got != "{}" {
		t.Fatalf("encode of something unencodable = %s", got)
	}
}

func TestEmptyArgumentsAreAnEmptyObject(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "  ", "null"} {
		if got := string(argumentsOf(json.RawMessage(raw))); got != "{}" {
			t.Fatalf("argumentsOf(%q) = %s", raw, got)
		}
	}
	if got := string(argumentsOf(json.RawMessage(`{"id":"p1"}`))); got != `{"id":"p1"}` {
		t.Fatalf("argumentsOf kept %s", got)
	}
}
