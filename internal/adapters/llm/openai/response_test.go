package openai_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/openai/openaitest"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestAnAnswerIsReadBackWhole(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		reply openaitest.Reply
		want  port.Response
	}{
		{
			name: "text with every counter",
			reply: openaitest.Answer{
				Text:  "Koffein und Powder",
				Usage: openaitest.Usage{Input: 1200, Cached: 1024, CacheWrite: 100, Output: 300, Reasoning: 200},
			}.Reply(),
			want: port.Response{
				Text:         "Koffein und Powder",
				Usage:        llm.Usage{Input: 1200, CachedInput: 1024, CacheWrite: 100, Output: 300, Reasoning: 200, Total: 1500},
				FinishReason: port.FinishStop,
				Tier:         llm.TierDefault,
			},
		},
		{
			name: "calls in the order they were made",
			reply: openaitest.Calls(
				openaitest.Call{ID: "call_1", Name: "pages_list", Arguments: `{"limit":5}`},
				openaitest.Call{ID: "call_2", Name: "pages_get", Arguments: `{"id":"p1"}`},
			).Reply(),
			want: port.Response{
				FinishReason: port.FinishStop,
				Calls: []port.ToolCall{
					{ID: "call_1", Name: "pages_list", Args: json.RawMessage(`{"limit":5}`)},
					{ID: "call_2", Name: "pages_get", Args: json.RawMessage(`{"id":"p1"}`)},
				},
				Tier: llm.TierDefault,
			},
		},
		{
			name:  "a call without arguments",
			reply: openaitest.Calls(openaitest.Call{ID: "call_1", Name: "sites_list"}).Reply(),
			want: port.Response{
				FinishReason: port.FinishStop,
				Calls:        []port.ToolCall{{ID: "call_1", Name: "sites_list", Args: json.RawMessage(`{}`)}},
				Tier:         llm.TierDefault,
			},
		},
		{
			name:  "a refusal",
			reply: openaitest.Refusal("I can't help with that.").Reply(),
			want:  port.Response{FinishReason: port.FinishContentFilter, Tier: llm.TierDefault},
		},
		{
			name:  "an answer cut at its ceiling",
			reply: openaitest.Truncated(`{"title":"Koff`).Reply(),
			want:  port.Response{Text: `{"title":"Koff`, FinishReason: port.FinishLength, Tier: llm.TierDefault},
		},
		{
			name:  "an answer stopped by the filter",
			reply: openaitest.Answer{Text: "half", Incomplete: "content_filter"}.Reply(),
			want:  port.Response{Text: "half", FinishReason: port.FinishContentFilter, Tier: llm.TierDefault},
		},
		{
			name:  "flex served",
			reply: openaitest.Answer{Text: "slow", Tier: "flex"}.Reply(),
			want:  port.Response{Text: "slow", FinishReason: port.FinishStop, Tier: llm.TierFlex},
		},
		{
			name:  "a tier we do not price",
			reply: openaitest.Answer{Text: "fast", Tier: "priority"}.Reply(),
			want:  port.Response{Text: "fast", FinishReason: port.FinishStop, Tier: llm.TierDefault},
		},
		{
			name:  "auto reported as the default",
			reply: openaitest.Answer{Text: "auto", Tier: "auto"}.Reply(),
			want:  port.Response{Text: "auto", FinishReason: port.FinishStop, Tier: llm.TierDefault},
		},
		{
			name: "a body that names no tier keeps the one sent",
			reply: openaitest.Reply{Status: http.StatusOK, Body: `{"status":"completed","output":[{"type":"message","content":[` +
				`{"type":"output_text","text":"a"},{"type":"output_text","text":"b"}]},{"type":"message","content":[{"type":"output_text","text":"c"}]}]}`},
			want: port.Response{Text: "abc", FinishReason: port.FinishStop, Tier: llm.TierDefault},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			server.Enqueue(tc.reply)

			got, err := newClient(server).Complete(t.Context(), write("hello"))
			if err != nil {
				t.Fatalf("Complete: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("response = %+v\nwant       %+v", got, tc.want)
			}
		})
	}
}

func TestAnAnswerTheClientCannotUseIsAFailure(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		body   string
		want   errors.Code
		reason string
	}{
		{name: "a body that is not json", body: `{"status":`, want: errors.External},
		{name: "an unfinished answer", body: `{"status":"in_progress","output":[]}`, want: errors.External},
		{
			name: "a call whose arguments are not json",
			body: `{"status":"completed","output":[{"type":"function_call","call_id":"call_1","name":"pages_get","arguments":"{\"id\":"}]}`,
			want: errors.External, reason: port.ReasonMalformedAnswer,
		},
		{
			name: "a call without an id",
			body: `{"status":"completed","output":[{"type":"function_call","name":"pages_get","arguments":"{}"}]}`,
			want: errors.External, reason: port.ReasonMalformedAnswer,
		},
		{
			name: "a failed answer",
			body: `{"status":"failed","error":{"code":"server_error","message":"The server had an error."},"output":[]}`,
			want: errors.External,
		},
		{
			name: "a failed answer that says the credit is gone",
			body: `{"status":"failed","error":{"code":"credit_balance_exhausted","message":"You have no credits remaining."},"output":[]}`,
			want: errors.NeedsHuman,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			server.Enqueue(openaitest.Reply{Status: http.StatusOK, Body: tc.body})

			_, err := newClient(server).Complete(t.Context(), write("hello"))
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("Complete = %v (%s), want %s", err, errors.CodeOf(err), tc.want)
			}
			if tc.reason != "" && kernelOf(t, err).Details["reason"] != tc.reason {
				t.Errorf("details = %v, want reason %s", kernelOf(t, err).Details, tc.reason)
			}
		})
	}
}

func roundTrip[T any](t *testing.T, answer string, want T) {
	t.Helper()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Text(answer).Reply())

	got, _, err := port.Structured[T](t.Context(), newClient(server), port.Request{
		Ref:      ref("gpt-5.6-terra"),
		Messages: []port.Message{{Role: port.RoleUser, Text: "answer"}},
		Meta:     port.CallMeta{Step: "round_trip"},
	})
	if err != nil {
		t.Fatalf("Structured: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("answer = %+v, want %+v", got, want)
	}
}

type judgeShape struct {
	Score       float64  `json:"score" description:"The overall quality of the page between 0 and 1"`
	Issues      []string `json:"issues"`
	Suggestions []string `json:"suggestions"`
}

type keywordShape struct {
	Keyword           string   `json:"keyword"`
	Kind              string   `json:"kind" enum:"hub,category,topic,product"`
	SecondaryKeywords []string `json:"secondaryKeywords"`
}

type keywordsShape struct {
	Entities []keywordShape `json:"entities"`
}

func TestEveryAnswerTheStepsAskForCrossesTheStrictContract(t *testing.T) {
	t.Parallel()

	t.Run("the writer's draft", func(t *testing.T) {
		t.Parallel()
		roundTrip(t, `{"title":"Koffein","h1":"Koffein Powder","sections":[{"slot":1,"heading":"Dose","html":"<p>3 g</p>"}],"summary":"Pure."}`,
			content.DraftAnswer{Title: "Koffein", H1: "Koffein Powder", Summary: "Pure.",
				Sections: []content.AnswerSection{{Slot: 1, Heading: "Dose", HTML: "<p>3 g</p>"}}})
	})
	t.Run("the writer's product", func(t *testing.T) {
		t.Parallel()
		roundTrip(t, `{"title":"Koffein","h1":"Koffein","sections":[],"summary":"s","shortDescription":"<p>short</p>",`+
			`"specifications":[{"name":"Purity","value":"99%"}]}`,
			content.ProductAnswer{
				DraftAnswer:      content.DraftAnswer{Title: "Koffein", H1: "Koffein", Sections: []content.AnswerSection{}, Summary: "s"},
				ShortDescription: "<p>short</p>",
				Specifications:   []content.AnswerSpecification{{Name: "Purity", Value: "99%"}},
			})
	})
	t.Run("the linker's repair", func(t *testing.T) {
		t.Parallel()
		roundTrip(t, `{"sentence":"Read about koffein powder."}`, content.RepairResponse{Sentence: "Read about koffein powder."})
	})
	t.Run("the judge's verdict", func(t *testing.T) {
		t.Parallel()
		roundTrip(t, `{"score":0.8,"issues":["thin"],"suggestions":[]}`, judgeShape{Score: 0.8, Issues: []string{"thin"}, Suggestions: []string{}})
	})
	t.Run("the keyword proposal", func(t *testing.T) {
		t.Parallel()
		roundTrip(t, `{"entities":[{"keyword":"koffein","kind":"topic","secondaryKeywords":["caffeine"]}]}`,
			keywordsShape{Entities: []keywordShape{{Keyword: "koffein", Kind: "topic", SecondaryKeywords: []string{"caffeine"}}}})
	})
}

type answerWithGaps struct {
	Words    *int      `json:"words"`
	Title    string    `json:"title"`
	Note     string    `json:"note,omitempty"`
	Tags     []string  `json:"tags,omitempty"`
	Owner    *section  `json:"owner,omitempty"`
	Sections []section `json:"sections"`
}

func TestAStructuredAnswerWithNullsDecodesToZeroValues(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Answer{
		Text:  `{"title":"Koffein","note":null,"words":null,"tags":null,"owner":null,"sections":[{"heading":"Dose","kind":"text"}]}`,
		Usage: openaitest.Usage{Input: 20, Output: 8},
	}.Reply())

	got, usage, err := port.Structured[answerWithGaps](t.Context(), newClient(server), port.Request{
		Ref:      ref("gpt-5.6-terra"),
		Messages: []port.Message{{Role: port.RoleUser, Text: "write the page"}},
		Meta:     port.CallMeta{Step: "generate_body"},
	})
	if err != nil {
		t.Fatalf("Structured: %v", err)
	}
	want := answerWithGaps{Title: "Koffein", Sections: []section{{Heading: "Dose", Kind: "text"}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("answer = %+v, want %+v", got, want)
	}
	if usage.Total != 28 {
		t.Errorf("usage = %+v, want 28 tokens", usage)
	}

	format, ok := only(t, server).Body["text"].(map[string]any)
	if !ok || format["format"] == nil {
		t.Fatalf("the request carried no format: %v", only(t, server).Body)
	}
}
