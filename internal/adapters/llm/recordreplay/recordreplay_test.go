package recordreplay_test

import (
	"encoding/json"
	stderrors "errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/llm/recordreplay"
	port "github.com/davidmovas/postulator/internal/application/llm"
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

func TestOffPassesThrough(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	client := recordreplay.New(fake.New(), "", "")
	if _, err := client.Complete(t.Context(), request("ANSWER: Koffein")); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	client = recordreplay.New(fake.New(), recordreplay.ModeOff, dir)
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
	recorder := recordreplay.New(inner, recordreplay.ModeRecord, dir)

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

	player := recordreplay.New(fake.New(), recordreplay.ModeReplay, dir)
	replayed, err := player.Complete(t.Context(), request("ANSWER: Koffein und Powder"))
	if err != nil {
		t.Fatalf("replay Complete: %v", err)
	}
	if replayed != recorded {
		t.Errorf("replayed = %+v, want %+v", replayed, recorded)
	}
}

func TestTheKeyIgnoresTheCallMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	recorder := recordreplay.New(fake.New(), recordreplay.ModeRecord, dir)
	if _, err := recorder.Complete(t.Context(), request("ANSWER: Koffein")); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	other := request("ANSWER: Koffein")
	other.Meta = port.CallMeta{RunID: "run-2", ItemID: "item-9", Step: "judge", ConversationID: "chat-3"}

	player := recordreplay.New(fake.New(), recordreplay.ModeReplay, dir)
	if _, err := player.Complete(t.Context(), other); err != nil {
		t.Fatalf("replay with other metadata: %v", err)
	}
}

func TestReplayWithoutAFixture(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	player := recordreplay.New(fake.New(), recordreplay.ModeReplay, dir)

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
	recorder := recordreplay.New(fake.New(), recordreplay.ModeRecord, dir)

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

	player := recordreplay.New(fake.New(), recordreplay.ModeReplay, dir)
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

func TestFailuresAreNotRecorded(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	recorder := recordreplay.New(fake.New(), recordreplay.ModeRecord, dir)

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
	recorder := recordreplay.New(fake.New(), recordreplay.ModeRecord, dir)
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

	player := recordreplay.New(fake.New(), recordreplay.ModeReplay, dir)
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
