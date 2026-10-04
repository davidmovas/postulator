package recordreplay_test

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/llm/recordreplay"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/tools"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

func request(prompt string) port.Request {
	return port.Request{
		Ref:      llm.ModelRef{Provider: "openai", Model: "gpt-5.6-luna"},
		System:   "you write pages",
		Messages: []port.Message{{Role: port.RoleUser, Text: prompt}},
		Meta:     port.CallMeta{RunID: "run-1", ItemID: "item-1"},
	}
}

func chatting(messages ...port.Message) port.Request {
	return port.Request{
		Ref:      llm.ModelRef{Provider: "openai", Model: "gpt-5.6-terra"},
		System:   "you run the site",
		Messages: messages,
		Tools:    []port.Tool{{Name: "pages_tree"}, {Name: "models_set_provider_key"}},
		Tier:     llm.TierFlex,
		Meta:     port.CallMeta{ConversationID: "chat-1", Step: "chat"},
	}
}

func asked(text string) port.Message {
	return port.Message{Role: port.RoleUser, Text: text}
}

func open(next port.Client, mode, dir string) *recordreplay.Client {
	return recordreplay.New(next, mode, dir, tools.Redact)
}

func TestOffPassesThrough(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := open(fake.New(), "", "")
	if _, err := client.Complete(t.Context(), request("ANSWER: Koffein")); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	client = open(fake.New(), recordreplay.ModeOff, dir)
	deltas, err := client.Stream(t.Context(), request("ANSWER: Koffein"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	for range deltas {
		continue
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the fixture directory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("fixtures = %v, want none while the decorator is off", entries)
	}
}

func TestRecordThenReplay(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "llm")
	inner := fake.New()
	recorder := open(inner, recordreplay.ModeRecord, dir)

	recorded, err := recorder.Complete(t.Context(), request("ANSWER: Koffein und Powder"))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the fixture directory: %v", err)
	}
	if len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), ".json") {
		t.Fatalf("fixtures = %v, want one json file", entries)
	}
	if len(strings.TrimSuffix(entries[0].Name(), ".json")) != 64 {
		t.Errorf("fixture name = %s, want a sha256 digest", entries[0].Name())
	}

	player := open(fake.New(), recordreplay.ModeReplay, dir)
	replayed, err := player.Complete(t.Context(), request("ANSWER: Koffein und Powder"))
	if err != nil {
		t.Fatalf("replay Complete: %v", err)
	}
	if !reflect.DeepEqual(replayed, recorded) {
		t.Errorf("replayed = %+v, want %+v", replayed, recorded)
	}
}

func TestTheKeyIgnoresTheCallMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	recorder := open(fake.New(), recordreplay.ModeRecord, dir)
	if _, err := recorder.Complete(t.Context(), request("ANSWER: Koffein")); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	other := request("ANSWER: Koffein")
	other.Meta = port.CallMeta{RunID: "run-2", ItemID: "item-9", Step: "judge", ConversationID: "chat-3"}

	player := open(fake.New(), recordreplay.ModeReplay, dir)
	if _, err := player.Complete(t.Context(), other); err != nil {
		t.Fatalf("replay with other metadata: %v", err)
	}
}

func TestReplayWithoutAFixture(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	player := open(fake.New(), recordreplay.ModeReplay, dir)

	_, err := player.Complete(t.Context(), request("ANSWER: missing"))
	if !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Complete error = %v, want %s", err, errors.NotFound)
	}

	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatalf("error is %T, want a kernel error", err)
	}
	dumped, ok := kernel.Details["request"].(string)
	if !ok || !strings.Contains(dumped, "ANSWER: missing") {
		t.Errorf("details = %v, want the request dumped", kernel.Details)
	}

	if _, err = player.Stream(t.Context(), request("ANSWER: missing")); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Stream error = %v, want %s", err, errors.NotFound)
	}
}

func TestStreamRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	recorder := open(fake.New(), recordreplay.ModeRecord, dir)

	deltas, err := recorder.Stream(t.Context(), request("ANSWER: Koffein und Powder"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	var recorded strings.Builder
	for delta := range deltas {
		if delta.Err != nil {
			t.Fatalf("delta error: %v", delta.Err)
		}
		recorded.WriteString(delta.Text)
	}
	if recorded.String() != "Koffein und Powder" {
		t.Fatalf("recorded = %q, want the streamed answer", recorded.String())
	}

	player := open(fake.New(), recordreplay.ModeReplay, dir)
	replayed, err := player.Stream(t.Context(), request("ANSWER: Koffein und Powder"))
	if err != nil {
		t.Fatalf("replay Stream: %v", err)
	}

	var text strings.Builder
	var done bool
	for delta := range replayed {
		text.WriteString(delta.Text)
		done = done || delta.Done
	}
	if !done || text.String() != "Koffein und Powder" {
		t.Errorf("replayed stream = %q, done %t", text.String(), done)
	}
}

type heard struct {
	calls  []port.ToolCall
	text   string
	usage  llm.Usage
	finish port.FinishReason
	tier   llm.ServiceTier
	done   bool
}

func hear(t *testing.T, deltas <-chan port.Delta) heard {
	t.Helper()

	var got heard
	for delta := range deltas {
		if delta.Err != nil {
			t.Fatalf("delta error: %v", delta.Err)
		}
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

func TestAConversationReplaysItsCallsAndItsTier(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		prompt string
		calls  int
		text   string
	}{
		{name: "a round that calls a tool", prompt: "TOOL:pages_tree{}\nFAKE: an empty tree", calls: 1},
		{name: "a round that answers", prompt: "FAKE: an empty tree", text: "an empty tree"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			req := chatting(asked(tc.prompt))

			deltas, err := open(fake.New(), recordreplay.ModeRecord, dir).Stream(t.Context(), req)
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}
			recorded := hear(t, deltas)

			replayed, err := open(fake.New(), recordreplay.ModeReplay, dir).Stream(t.Context(), req)
			if err != nil {
				t.Fatalf("replay Stream: %v", err)
			}
			again := hear(t, replayed)

			if !reflect.DeepEqual(again, recorded) {
				t.Fatalf("replayed %+v, want what was recorded %+v", again, recorded)
			}
			if len(again.calls) != tc.calls || again.text != tc.text || again.tier != llm.TierFlex || !again.done {
				t.Fatalf("the replay is %+v", again)
			}

			answered, err := open(fake.New(), recordreplay.ModeReplay, dir).Complete(t.Context(), req)
			if err != nil {
				t.Fatalf("replay Complete: %v", err)
			}
			if len(answered.Calls) != tc.calls || answered.Text != tc.text || answered.Tier != llm.TierFlex {
				t.Fatalf("the replayed response is %+v", answered)
			}
		})
	}
}

func TestACredentialInAToolCallNeverReachesAFixture(t *testing.T) {
	t.Parallel()

	const (
		spoken  = "sk-live-SpokenSpokenSpoken1234"
		history = "sk-live-HistoryHistoryHistory5678"
	)

	earlier := port.ToolCall{
		ID: "call-1", Name: "models_set_provider_key",
		Args: json.RawMessage(`{"provider":"openai","apiKey":"` + history + `"}`),
	}
	req := chatting(
		asked("store my key"),
		port.Message{Role: port.RoleAssistant, Call: &earlier},
		port.Message{Role: port.RoleTool, Result: &port.ToolResult{CallID: "call-1", Output: json.RawMessage(`{"ok":true}`)}},
		asked(`TOOL:models_set_provider_key{"provider":"openai","apiKey":"`+spoken+`"}`+"\nFAKE: stored"),
	)

	for _, mode := range []string{"complete", "stream"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			recorder := open(fake.New(), recordreplay.ModeRecord, dir)
			if mode == "complete" {
				if _, err := recorder.Complete(t.Context(), req); err != nil {
					t.Fatalf("Complete: %v", err)
				}
			} else {
				deltas, err := recorder.Stream(t.Context(), req)
				if err != nil {
					t.Fatalf("Stream: %v", err)
				}
				hear(t, deltas)
			}

			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 {
				t.Fatalf("fixtures = %v, %v", entries, err)
			}
			written, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
			if err != nil {
				t.Fatalf("read the fixture: %v", err)
			}
			if strings.Contains(string(written), history) {
				t.Fatalf("the fixture carries the key an earlier call was given: %s", written)
			}
			if !strings.Contains(string(written), `"***"`) {
				t.Fatalf("the fixture does not say a key was there: %s", written)
			}

			replayed, err := open(fake.New(), recordreplay.ModeReplay, dir).Complete(t.Context(), req)
			if err != nil {
				t.Fatalf("the replay of the unmasked request found no fixture: %v", err)
			}
			if len(replayed.Calls) != 1 || strings.Contains(string(replayed.Calls[0].Args), spoken) {
				t.Fatalf("the replayed call is %+v", replayed.Calls)
			}
		})
	}
}

type searching struct{}

func (searching) answer() port.Response {
	return port.Response{
		Searches: []port.ToolSearch{
			{Kind: port.SearchCall, Execution: "server", Payload: json.RawMessage(`{"paths":["pages"]}`)},
			{Kind: port.SearchOutput, Execution: "server", Payload: json.RawMessage(`[{"type":"namespace","name":"pages"}]`)},
		},
		Calls: []port.ToolCall{{
			ID: "call-1", Name: "models_set_provider_key", Namespace: "models",
			Args: json.RawMessage(`{"provider":"openai","apiKey":"sk-live-SearchedSearched9876"}`),
		}},
		Usage:        llm.Usage{Input: 900, Output: 40, Total: 940},
		FinishReason: port.FinishStop,
	}
}

func (s searching) Complete(context.Context, port.Request) (port.Response, error) {
	return s.answer(), nil
}

func (s searching) Stream(context.Context, port.Request) (<-chan port.Delta, error) {
	resp := s.answer()
	out := make(chan port.Delta, len(resp.Searches)+len(resp.Calls)+1)
	for i := range resp.Searches {
		out <- port.Delta{Search: &resp.Searches[i]}
	}
	for i := range resp.Calls {
		out <- port.Delta{Call: &resp.Calls[i]}
	}
	out <- port.Delta{Done: true, Usage: &resp.Usage, Finish: resp.FinishReason}
	close(out)
	return out, nil
}

func compacted(t *testing.T, searches []port.ToolSearch) []port.ToolSearch {
	t.Helper()

	out := make([]port.ToolSearch, 0, len(searches))
	for _, search := range searches {
		var dense bytes.Buffer
		if err := json.Compact(&dense, search.Payload); err != nil {
			t.Fatalf("compact %s: %v", search.Payload, err)
		}
		search.Payload = dense.Bytes()
		out = append(out, search)
	}
	return out
}

type step struct {
	search *port.ToolSearch
	call   *port.ToolCall
}

func steps(t *testing.T, deltas <-chan port.Delta) []step {
	t.Helper()

	var got []step
	for delta := range deltas {
		if delta.Err != nil {
			t.Fatalf("delta error: %v", delta.Err)
		}
		switch {
		case delta.Search != nil:
			got = append(got, step{search: delta.Search})
		case delta.Call != nil:
			got = append(got, step{call: delta.Call})
		}
	}
	return got
}

func TestADeferredToolSearchIsRecordedAndReplayedBeforeTheCallItLoaded(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"complete", "stream"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			req := chatting(asked("read the page"))
			recorder := open(searching{}, recordreplay.ModeRecord, dir)
			if mode == "complete" {
				if _, err := recorder.Complete(t.Context(), req); err != nil {
					t.Fatalf("Complete: %v", err)
				}
			} else {
				deltas, err := recorder.Stream(t.Context(), req)
				if err != nil {
					t.Fatalf("Stream: %v", err)
				}
				steps(t, deltas)
			}

			answered, err := open(fake.New(), recordreplay.ModeReplay, dir).Complete(t.Context(), req)
			if err != nil {
				t.Fatalf("replay Complete: %v", err)
			}
			if got, want := compacted(t, answered.Searches), compacted(t, searching{}.answer().Searches); !reflect.DeepEqual(got, want) {
				t.Fatalf("replayed searches = %+v, want %+v", got, want)
			}

			replayed, err := open(fake.New(), recordreplay.ModeReplay, dir).Stream(t.Context(), req)
			if err != nil {
				t.Fatalf("replay Stream: %v", err)
			}
			got := steps(t, replayed)
			if len(got) != 3 || got[0].search == nil || got[1].search == nil || got[2].call == nil {
				t.Fatalf("replayed stream = %+v, want the search call, its output, then the call", got)
			}
			if got[0].search.Kind != port.SearchCall || got[1].search.Kind != port.SearchOutput {
				t.Fatalf("replayed searches = %+v then %+v, want the call before its output", *got[0].search, *got[1].search)
			}
			if got[2].call.Namespace != "models" || strings.Contains(string(got[2].call.Args), "SearchedSearched") {
				t.Fatalf("replayed call = %+v, want its namespace and its key masked", *got[2].call)
			}
		})
	}
}

func TestAStreamThatNeverEndsIsNotRecorded(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	recorder := open(unfinished{}, recordreplay.ModeRecord, dir)

	deltas, err := recorder.Stream(t.Context(), request("ANSWER: half"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	for range deltas {
		continue
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the fixture directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("fixtures = %v, want none for a stream that never finished", entries)
	}
}

type unfinished struct{}

func (unfinished) Complete(context.Context, port.Request) (port.Response, error) {
	return port.Response{}, errors.New(errors.Internal, "the unfinished client never completes")
}

func (unfinished) Stream(context.Context, port.Request) (<-chan port.Delta, error) {
	out := make(chan port.Delta, 1)
	out <- port.Delta{Text: "half"}
	close(out)
	return out, nil
}

func TestFailuresAreNotRecorded(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	recorder := open(fake.New(), recordreplay.ModeRecord, dir)

	if _, err := recorder.Complete(t.Context(), request("ERROR: EXTERNAL")); !errors.IsCode(err, errors.External) {
		t.Fatalf("Complete error = %v, want %s", err, errors.External)
	}
	if _, err := recorder.Stream(t.Context(), request("ERROR: EXTERNAL")); !errors.IsCode(err, errors.External) {
		t.Fatalf("Stream error = %v, want %s", err, errors.External)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the fixture directory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("fixtures = %v, want none for failed calls", entries)
	}
}

func TestCorruptFixture(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	recorder := open(fake.New(), recordreplay.ModeRecord, dir)
	if _, err := recorder.Complete(t.Context(), request("ANSWER: Koffein")); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the fixture directory: %v", err)
	}
	if err = os.WriteFile(filepath.Join(dir, entries[0].Name()), []byte("{"), 0o600); err != nil {
		t.Fatalf("corrupt the fixture: %v", err)
	}

	player := open(fake.New(), recordreplay.ModeReplay, dir)
	if _, err = player.Complete(t.Context(), request("ANSWER: Koffein")); !errors.IsCode(err, errors.Internal) {
		t.Fatalf("Complete error = %v, want %s", err, errors.Internal)
	}
}

func TestModeSetting(t *testing.T) {
	t.Parallel()

	values := settings.Default().NewValues()
	if got := recordreplay.Mode(values); got != recordreplay.ModeOff {
		t.Fatalf("Mode = %s, want %s", got, recordreplay.ModeOff)
	}

	encoded, err := json.Marshal(recordreplay.ModeReplay)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if _, err = settings.Default().Apply(values, map[string]json.RawMessage{"llm.recordReplayMode": encoded}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got := recordreplay.Mode(values); got != recordreplay.ModeReplay {
		t.Errorf("Mode = %s, want %s", got, recordreplay.ModeReplay)
	}

	bad, err := json.Marshal("sometimes")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if _, err = settings.Default().Apply(values, map[string]json.RawMessage{"llm.recordReplayMode": bad}); err == nil {
		t.Error("Apply accepted an unknown mode")
	}
}
