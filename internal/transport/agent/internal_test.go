package agent

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gollem-dev/gollem"
	"github.com/sashabaranov/go-openai"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	applicationllm "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func message(t *testing.T, role gollem.MessageRole, text string) gollem.Message {
	t.Helper()

	content, err := gollem.NewTextContent(text)
	if err != nil {
		t.Fatalf("NewTextContent: %v", err)
	}
	return gollem.Message{Role: role, Contents: []gollem.MessageContent{content}}
}

func toolResponse(t *testing.T, name string) gollem.Message {
	t.Helper()

	content, err := gollem.NewToolResponseContent("call-1", name, map[string]any{"ok": true}, false)
	if err != nil {
		t.Fatalf("NewToolResponseContent: %v", err)
	}
	return gollem.Message{Role: gollem.RoleUser, Contents: []gollem.MessageContent{content}}
}

func TestTrimKeepsTheNewestWholeTurns(t *testing.T) {
	t.Parallel()

	history := &gollem.History{
		LLType:  gollem.LLMTypeOpenAI,
		Version: gollem.HistoryVersion,
		Messages: []gollem.Message{
			message(t, gollem.RoleUser, strings.Repeat("a", 400)),
			message(t, gollem.RoleAssistant, strings.Repeat("b", 400)),
			message(t, gollem.RoleUser, "the newest question"),
			toolResponse(t, "pages_tree"),
			message(t, gollem.RoleAssistant, "the newest answer"),
		},
	}

	if kept := trim(history, 100000); len(kept.Messages) != 5 {
		t.Fatalf("a history inside its budget is kept whole: %d messages", len(kept.Messages))
	}

	trimmed := trim(history, 300)
	if len(trimmed.Messages) != 3 || trimmed.Messages[0].Role != gollem.RoleUser {
		t.Fatalf("the trimmed history is %+v", trimmed.Messages)
	}
	if size(trimmed) >= size(history) {
		t.Fatalf("the trimmed history is %d bytes of %d", size(trimmed), size(history))
	}
	if size(trim(history, 800)) > 800 {
		t.Fatal("a turn that fits the budget is kept whole")
	}

	if trim(nil, 10) != nil {
		t.Error("trimming nothing answers nothing")
	}
	empty := &gollem.History{Version: gollem.HistoryVersion}
	if kept := trim(empty, 10); len(kept.Messages) != 0 {
		t.Error("trimming an empty history answers it unchanged")
	}
}

const fenceBytes = 64

type heldHistory struct {
	body    []byte
	version int
}

func (h *heldHistory) Load(context.Context, string) (body []byte, version int, err error) {
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

func fatResponse(t *testing.T, name string, rows int) gollem.Message {
	t.Helper()

	items := make([]any, 0, rows)
	for i := range rows {
		items = append(items, map[string]any{
			"id":    fmt.Sprintf("page-%03d", i),
			"path":  fmt.Sprintf("/coffee/espresso/%03d/", i),
			"title": strings.Repeat("Espresso ", 8),
		})
	}

	content, err := gollem.NewToolResponseContent("call-1", name,
		agentapp.Fence(map[string]any{"items": items, "hasMore": true, "nextCursor": "c-42"}), false)
	if err != nil {
		t.Fatalf("NewToolResponseContent: %v", err)
	}
	return gollem.Message{Role: gollem.RoleUser, Contents: []gollem.MessageContent{content}}
}

func TestAStoredToolResultReplaysShortenedAndStillDecodes(t *testing.T) {
	t.Parallel()

	fat := fatResponse(t, "pages_list", 128)
	if measured := len(fat.Contents[0].Data); measured < agentapp.DefaultMaxToolResultBytes {
		t.Fatalf("the fixture is %d bytes, want at least the in-turn ceiling", measured)
	}

	store := &heldHistory{}
	held := bounded{
		store: store, clock: clock.NewFake(time.Date(2026, 9, 22, 9, 0, 0, 0, time.UTC)),
		budget: agentapp.DefaultHistoryBudgetChars,
		cap:    agentapp.DefaultHistoryToolResultBytes,
	}

	written := &gollem.History{
		LLType:  gollem.LLMTypeOpenAI,
		Version: gollem.HistoryVersion,
		Messages: []gollem.Message{
			message(t, gollem.RoleUser, "list the espresso pages"),
			fat,
			message(t, gollem.RoleAssistant, "a hundred and twenty pages"),
		},
	}
	if err := held.Save(t.Context(), "c1", written); err != nil {
		t.Fatalf("Save: %v", err)
	}

	replayed, err := held.Load(t.Context(), "c1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(replayed.Messages) != 3 {
		t.Fatalf("the replayed history holds %d messages", len(replayed.Messages))
	}
	if replayed.Messages[0].Contents[0].Type != gollem.MessageContentTypeText {
		t.Fatal("a text message must be replayed untouched")
	}

	shortened := replayed.Messages[1].Contents[0]
	if len(shortened.Data) >= len(fat.Contents[0].Data) {
		t.Fatalf("the stored result is %d bytes of the original %d", len(shortened.Data), len(fat.Contents[0].Data))
	}
	if len(shortened.Data) > agentapp.DefaultHistoryToolResultBytes+fenceBytes {
		t.Fatalf("the stored result is %d bytes, over the history ceiling", len(shortened.Data))
	}

	answered, err := shortened.GetToolResponseContent()
	if err != nil {
		t.Fatalf("the shortened result no longer decodes: %v", err)
	}
	if answered.ToolCallID != "call-1" || answered.Name != "pages_list" {
		t.Fatalf("the shortened result lost its call: %+v", answered)
	}
	if answered.Response[agentapp.UntrustedMarker] != true {
		t.Fatalf("the shortened result lost its fence: %+v", answered.Response)
	}

	inner, ok := answered.Response[agentapp.UntrustedData].(map[string]any)
	if !ok || inner[agentapp.TruncatedKey] != true {
		t.Fatalf("the shortened result does not say it was shortened: %+v", answered.Response)
	}
	document, ok := inner[agentapp.ResultKey].(map[string]any)
	if !ok || document["nextCursor"] != "c-42" || document["hasMore"] != true {
		t.Fatalf("the shortened result lost the fields that say how to ask for the rest: %+v", inner)
	}
	kept, ok := document["items"].([]any)
	if !ok || len(kept) == 0 || len(kept) >= 128 {
		t.Fatalf("the shortened result kept %d of 128 rows", len(kept))
	}
}

func TestAResultThatFitsIsStoredUntouched(t *testing.T) {
	t.Parallel()

	small := toolResponse(t, "pages_tree")
	history := &gollem.History{
		LLType:   gollem.LLMTypeOpenAI,
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{message(t, gollem.RoleUser, "hello"), small},
	}

	kept := shorten(history, agentapp.DefaultHistoryToolResultBytes)
	if string(kept.Messages[1].Contents[0].Data) != string(small.Contents[0].Data) {
		t.Fatalf("a result inside the ceiling was rewritten: %s", kept.Messages[1].Contents[0].Data)
	}
	if shorten(nil, 100) != nil {
		t.Error("shortening nothing answers nothing")
	}
	if shorten(history, 0) != history {
		t.Error("no ceiling shortens nothing")
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

			runner := New(Deps{}, Config{})
			if got := runner.historyCeiling(tc.spec); got != tc.want {
				t.Fatalf("historyCeiling = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestTrimDropsAHistoryThatStartsWithNoTurn(t *testing.T) {
	t.Parallel()

	history := &gollem.History{
		LLType:   gollem.LLMTypeClaude,
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{message(t, gollem.RoleAssistant, strings.Repeat("a", 500))},
	}

	trimmed := trim(history, 100)
	if len(trimmed.Messages) != 0 || trimmed.LLType != gollem.LLMTypeClaude {
		t.Fatalf("the trimmed history is %+v", trimmed)
	}
}

func TestObjectOfWrapsWhatIsNotAnObject(t *testing.T) {
	t.Parallel()

	object, err := objectOf(map[string]any{"id": "p1"})
	if err != nil || object["id"] != "p1" {
		t.Fatalf("objectOf = %v, %v", object, err)
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

func TestParametersFollowTheSchema(t *testing.T) {
	t.Parallel()

	schema := &applicationllm.Schema{
		Type:     applicationllm.SchemaObject,
		Required: []string{"path"},
		Properties: map[string]*applicationllm.Schema{
			"path":  {Type: applicationllm.SchemaString, Description: "the path"},
			"tags":  {Type: applicationllm.SchemaArray, Items: &applicationllm.Schema{Type: applicationllm.SchemaString}},
			"loose": {Type: applicationllm.SchemaArray},
			"nested": {
				Type:     applicationllm.SchemaObject,
				Required: []string{"id"},
				Properties: map[string]*applicationllm.Schema{
					"id":   {Type: applicationllm.SchemaString},
					"hint": {Type: applicationllm.SchemaString},
				},
			},
			"rows": {
				Type: applicationllm.SchemaArray,
				Items: &applicationllm.Schema{
					Type:     applicationllm.SchemaObject,
					Required: []string{"text"},
					Properties: map[string]*applicationllm.Schema{
						"text":   {Type: applicationllm.SchemaString},
						"weight": {Type: applicationllm.SchemaNumber},
					},
				},
			},
		},
	}

	parameters := parametersOf(schema)
	if !parameters["path"].Required || parameters["tags"].Required {
		t.Fatalf("the required flags are %+v", parameters)
	}
	if parameters["tags"].Items == nil || parameters["loose"].Items == nil {
		t.Fatal("an array parameter always carries an item type, because gollem refuses one without")
	}
	if parameters["nested"].Properties["id"] == nil || !parameters["nested"].Properties["id"].Required {
		t.Fatalf("the nested parameter is %+v", parameters["nested"])
	}
	if parameters["nested"].Properties["hint"].Required {
		t.Fatalf("a nested field its own schema leaves out was still required: %+v", parameters["nested"])
	}
	if parameters["rows"].Items == nil || !parameters["rows"].Items.Properties["text"].Required {
		t.Fatalf("the item parameter is %+v", parameters["rows"].Items)
	}
	if parameters["rows"].Items.Properties["weight"].Required {
		t.Fatalf("an optional field of a list item was still required: %+v", parameters["rows"].Items)
	}
	if len(parametersOf(nil)) != 0 {
		t.Error("a tool with no schema takes no parameters")
	}
	if parameterOf(nil).Type != gollem.TypeString {
		t.Error("a missing property schema falls back to a string")
	}
}

func TestConvertNamesTheFailure(t *testing.T) {
	t.Parallel()

	if convert(nil) != nil {
		t.Error("convert of nothing is nothing")
	}
	if err := convert(errors.New(errors.NotFound, "gone")); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("a coded error passes through: %v", err)
	}
	if err := convert(context.Canceled); !errors.IsCode(err, errors.Cancelled) {
		t.Errorf("a cancelled turn = %v", err)
	}
	if err := convert(gollem.ErrLoopLimitExceeded); !errors.IsCode(err, errors.BudgetExceeded) {
		t.Errorf("an exhausted loop = %v", err)
	}
	overflow := convert(fmt.Errorf("session: %w", gollem.ErrTokenSizeExceeded))
	if !errors.IsCode(overflow, errors.Invalid) || !strings.Contains(overflow.Error(), "context window") {
		t.Errorf("a turn that outgrew the context window = %v", overflow)
	}
	if err := convert(json.Unmarshal([]byte("x"), &struct{}{})); !errors.IsCode(err, errors.External) {
		t.Errorf("an unknown model failure = %v", err)
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

func TestConvertSaysWhatTheProviderSaid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		err     error
		want    errors.Code
		message string
	}{
		{
			name:    "a key without access to the model",
			err:     &openai.APIError{HTTPStatusCode: http.StatusForbidden, Message: "the project cannot reach this model"},
			want:    errors.Unauthorized,
			message: "the project cannot reach this model",
		},
		{
			name:    "a model the provider does not have",
			err:     &openai.APIError{HTTPStatusCode: http.StatusNotFound, Message: "unknown model"},
			want:    errors.NotFound,
			message: "unknown model",
		},
		{
			name:    "a rejected request",
			err:     &openai.APIError{HTTPStatusCode: http.StatusBadRequest, Message: "tools[7].function.parameters is invalid"},
			want:    errors.Invalid,
			message: "tools[7].function.parameters is invalid",
		},
		{
			name: "a rate limited key",
			err:  &openai.APIError{HTTPStatusCode: http.StatusTooManyRequests, Message: "slow down"},
			want: errors.RateLimited,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := convert(tc.err)
			if !errors.IsCode(got, tc.want) {
				t.Fatalf("convert = %v (%s), want %s", got, errors.CodeOf(got), tc.want)
			}
			if tc.message == "" {
				return
			}

			var kernel *errors.Error
			if !stderrors.As(got, &kernel) {
				t.Fatalf("convert returned %T, want a kernel error", got)
			}
			if kernel.Details["providerMessage"] != tc.message {
				t.Errorf("providerMessage = %v, want %q", kernel.Details["providerMessage"], tc.message)
			}
		})
	}
}
