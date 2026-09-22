package agent

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gollem-dev/gollem"
	"github.com/sashabaranov/go-openai"

	applicationllm "github.com/davidmovas/postulator/internal/application/llm"
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
