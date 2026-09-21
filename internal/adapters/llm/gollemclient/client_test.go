package gollemclient_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/gollemclient"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

type vault map[string]string

func (v vault) Get(_ context.Context, ref string) (string, error) {
	value, ok := v[ref]
	if !ok {
		return "", errors.New(errors.NotFound, "no secret is stored under this reference")
	}
	return value, nil
}

func newValues(t *testing.T, overrides map[string]string) *settings.Values {
	t.Helper()

	stored := make(map[string]json.RawMessage, len(overrides))
	for key, value := range overrides {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("encode %s: %v", key, err)
		}
		stored[key] = encoded
	}

	values := settings.Default().NewValues()
	if _, err := settings.Default().Apply(values, stored); err != nil {
		t.Fatalf("apply the settings: %v", err)
	}
	return values
}

const (
	openaiModel = "gpt-5.6-terra"
	claudeModel = "claude-sonnet-5"
)

func openaiRef() llm.ModelRef {
	return llm.ModelRef{Provider: gollemclient.ProviderOpenAI, Model: openaiModel}
}

func claudeRef() llm.ModelRef {
	return llm.ModelRef{Provider: gollemclient.ProviderAnthropic, Model: claudeModel}
}

type catalog map[string]llm.ModelInfo

func (c catalog) Lookup(_ context.Context, ref llm.ModelRef) (llm.ModelInfo, error) {
	info, ok := c[ref.String()]
	if !ok {
		return llm.ModelInfo{}, errors.New(errors.NotFound, "the catalog has no such model")
	}
	return info, nil
}

func newClient(t *testing.T, provider, setting string, handler http.HandlerFunc) (*gollemclient.Client, *bodies) {
	t.Helper()

	return newClientOver(t, provider, setting, nil, handler)
}

func newClientOver(t *testing.T, provider, setting string, models gollemclient.ModelReader, handler http.HandlerFunc) (*gollemclient.Client, *bodies) {
	t.Helper()

	captured := &bodies{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read the request body: %v", err)
		}
		captured.add(string(body))
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	values := newValues(t, map[string]string{setting: server.URL})
	factory := gollemclient.NewFactory(vault{gollemclient.SecretRef(provider): "test-key"}, nil, values)
	return gollemclient.New(factory, models, 10*time.Second), captured
}

type bodies struct {
	items []string
}

func (b *bodies) add(body string) {
	b.items = append(b.items, body)
}

func (b *bodies) last() string {
	if len(b.items) == 0 {
		return ""
	}
	return b.items[len(b.items)-1]
}

func writeJSON(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if _, err := io.WriteString(w, body); err != nil {
		t.Errorf("write the response: %v", err)
	}
}

func writeSSE(t *testing.T, w http.ResponseWriter, events []string) {
	t.Helper()

	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	flusher, ok := w.(http.Flusher)
	if !ok {
		t.Fatal("the test server does not flush")
	}
	for _, event := range events {
		if _, err := io.WriteString(w, event); err != nil {
			t.Errorf("write the event: %v", err)
			return
		}
		flusher.Flush()
	}
}

const openaiCompletion = `{"id":"cmpl-1","object":"chat.completion","created":1,"model":"gpt-5.6-terra",
	"choices":[{"index":0,"message":{"role":"assistant","content":"Koffein und Powder"},"finish_reason":"stop"}],
	"usage":{"prompt_tokens":12,"completion_tokens":4,"total_tokens":16}}`

const claudeCompletion = `{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-5",
	"content":[{"type":"text","text":"Koffein und Powder"}],"stop_reason":"end_turn","stop_sequence":null,
	"usage":{"input_tokens":12,"output_tokens":4}}`

func TestComplete(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		provider string
		setting  string
		ref      llm.ModelRef
		body     string
	}{
		{
			name:     "openai",
			provider: gollemclient.ProviderOpenAI,
			setting:  "llm.openai.baseUrl",
			ref:      openaiRef(),
			body:     openaiCompletion,
		},
		{
			name:     "anthropic",
			provider: gollemclient.ProviderAnthropic,
			setting:  "llm.anthropic.baseUrl",
			ref:      claudeRef(),
			body:     claudeCompletion,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client, captured := newClient(t, tc.provider, tc.setting, func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(t, w, tc.body)
			})

			resp, err := client.Complete(t.Context(), port.Request{
				Ref:       tc.ref,
				System:    "you write pages",
				Messages:  []port.Message{{Role: port.RoleUser, Text: "hello"}, {Role: port.RoleAssistant, Text: "hi"}, {Role: port.RoleUser, Text: "write"}},
				MaxTokens: 256,
			})
			if err != nil {
				t.Fatalf("Complete: %v", err)
			}
			if resp.Text != "Koffein und Powder" {
				t.Errorf("text = %q, want the provider answer", resp.Text)
			}
			if (resp.Usage != llm.Usage{Input: 12, Output: 4, Total: 16}) {
				t.Errorf("usage = %+v, want 12 in and 4 out", resp.Usage)
			}
			if resp.FinishReason != port.FinishStop {
				t.Errorf("finish reason = %s, want %s", resp.FinishReason, port.FinishStop)
			}

			sent := captured.last()
			for _, want := range []string{"you write pages", "hello", "hi", "write", tc.ref.Model} {
				if !strings.Contains(sent, want) {
					t.Errorf("the request body does not carry %q: %s", want, sent)
				}
			}
		})
	}
}

func TestCompleteReportsTheLengthCeiling(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, gollemclient.ProviderOpenAI, "llm.openai.baseUrl", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, openaiCompletion)
	})

	resp, err := client.Complete(t.Context(), port.Request{
		Ref:       openaiRef(),
		Messages:  []port.Message{{Role: port.RoleUser, Text: "write"}},
		MaxTokens: 4,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.FinishReason != port.FinishLength {
		t.Errorf("finish reason = %s, want %s", resp.FinishReason, port.FinishLength)
	}
}

var openaiStream = []string{
	"data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-5.6-terra\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Koffein\"},\"finish_reason\":null}]}\n\n",
	"data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-5.6-terra\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" und Powder\"},\"finish_reason\":null}]}\n\n",
	"data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-5.6-terra\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
	"data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-5.6-terra\",\"choices\":[],\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":4,\"total_tokens\":16}}\n\n",
	"data: [DONE]\n\n",
}

func TestStream(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		provider string
		setting  string
		ref      llm.ModelRef
		events   []string
	}{
		{name: "openai", provider: gollemclient.ProviderOpenAI, setting: "llm.openai.baseUrl", ref: openaiRef(), events: openaiStream},
		{name: "anthropic", provider: gollemclient.ProviderAnthropic, setting: "llm.anthropic.baseUrl", ref: claudeRef()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client, _ := newClient(t, tc.provider, tc.setting, func(w http.ResponseWriter, _ *http.Request) {
				if tc.events == nil {
					writeJSON(t, w, claudeCompletion)
					return
				}
				writeSSE(t, w, tc.events)
			})

			deltas, err := client.Stream(t.Context(), port.Request{
				Ref:      tc.ref,
				Messages: []port.Message{{Role: port.RoleUser, Text: "write"}},
			})
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}

			var (
				text  strings.Builder
				usage llm.Usage
				done  bool
			)
			for delta := range deltas {
				if delta.Err != nil {
					t.Fatalf("delta error: %v", delta.Err)
				}
				text.WriteString(delta.Text)
				if delta.Done {
					done = true
					if delta.Usage != nil {
						usage = *delta.Usage
					}
				}
			}

			if !done {
				t.Error("the stream never reported done")
			}
			if text.String() != "Koffein und Powder" {
				t.Errorf("text = %q, want the streamed answer", text.String())
			}
			if usage.Input != 12 || usage.Output != 4 {
				t.Errorf("usage = %+v, want 12 in and 4 out", usage)
			}
		})
	}
}

type meta struct {
	Title       string `json:"title" description:"the page title"`
	Description string `json:"description"`
}

func TestStructuredOverTheProvider(t *testing.T) {
	t.Parallel()

	const answer = `{"id":"cmpl-1","object":"chat.completion","created":1,"model":"gpt-5.6-terra",
		"choices":[{"index":0,"message":{"role":"assistant","content":"{\"title\":\"Koffein\",\"description\":\"pure\"}"},"finish_reason":"stop"}],
		"usage":{"prompt_tokens":20,"completion_tokens":8,"total_tokens":28}}`

	client, captured := newClient(t, gollemclient.ProviderOpenAI, "llm.openai.baseUrl", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, answer)
	})

	got, usage, err := port.Structured[meta](t.Context(), client, port.Request{
		Ref:      openaiRef(),
		Messages: []port.Message{{Role: port.RoleUser, Text: "write the meta"}},
	})
	if err != nil {
		t.Fatalf("Structured: %v", err)
	}
	if got.Title != "Koffein" || got.Description != "pure" {
		t.Errorf("meta = %+v, want the decoded answer", got)
	}
	if usage.Total != 28 {
		t.Errorf("usage = %+v, want 28 tokens", usage)
	}

	sent := captured.last()
	for _, want := range []string{"json_schema", "the page title", "description"} {
		if !strings.Contains(sent, want) {
			t.Errorf("the request body does not carry %q: %s", want, sent)
		}
	}
}

func TestClientRejectsAnInvalidRequest(t *testing.T) {
	t.Parallel()

	client := gollemclient.New(gollemclient.NewFactory(vault{}, nil, newValues(t, nil)), nil, time.Second)
	req := port.Request{Ref: llm.ModelRef{}, Messages: nil}

	if _, err := client.Complete(t.Context(), req); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("Complete error = %v, want %s", err, errors.Invalid)
	}
	if _, err := client.Stream(t.Context(), req); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("Stream error = %v, want %s", err, errors.Invalid)
	}
}

func TestProviderErrorsAreClassified(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status int
		want   errors.Code
	}{
		{name: "a rejected key", status: http.StatusUnauthorized, want: errors.Unauthorized},
		{name: "an unknown model", status: http.StatusNotFound, want: errors.NotFound},
		{name: "a bad request", status: http.StatusBadRequest, want: errors.Invalid},
		{name: "a rate limit", status: http.StatusTooManyRequests, want: errors.RateLimited},
		{name: "a server error", status: http.StatusInternalServerError, want: errors.External},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client, _ := newClient(t, gollemclient.ProviderOpenAI, "llm.openai.baseUrl", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				writeJSON(t, w, `{"error":{"message":"no","type":"invalid_request_error"}}`)
			})

			_, err := client.Complete(t.Context(), port.Request{
				Ref:      openaiRef(),
				Messages: []port.Message{{Role: port.RoleUser, Text: "write"}},
			})
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("Complete error = %v (%s), want %s", err, errors.CodeOf(err), tc.want)
			}
		})
	}
}

func TestCancelledCallsAreNotRetryable(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, gollemclient.ProviderOpenAI, "llm.openai.baseUrl", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, openaiCompletion)
	})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := client.Complete(ctx, port.Request{Ref: openaiRef(), Messages: []port.Message{{Role: port.RoleUser, Text: "write"}}})
	if !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("Complete error = %v (%s), want %s", err, errors.CodeOf(err), errors.Cancelled)
	}
}

func TestFactoryRefusals(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		ref     llm.ModelRef
		secrets vault
		values  map[string]string
		want    errors.Code
	}{
		{name: "an empty reference", ref: llm.ModelRef{}, want: errors.Invalid},
		{name: "an unknown provider", ref: llm.ModelRef{Provider: "cohere", Model: "x"}, want: errors.Invalid},
		{name: "no stored key", ref: openaiRef(), want: errors.Unauthorized},
		{
			name:    "an empty stored key",
			ref:     openaiRef(),
			secrets: vault{gollemclient.SecretRef(gollemclient.ProviderOpenAI): ""},
			want:    errors.Unauthorized,
		},
		{name: "gemini without a project", ref: llm.ModelRef{Provider: gollemclient.ProviderGemini, Model: "gemini-3.5-flash"}, want: errors.Invalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			secrets := tc.secrets
			if secrets == nil {
				secrets = vault{}
			}
			factory := gollemclient.NewFactory(secrets, nil, newValues(t, tc.values))
			if _, err := factory.New(t.Context(), tc.ref); !errors.IsCode(err, tc.want) {
				t.Fatalf("New error = %v (%s), want %s", err, errors.CodeOf(err), tc.want)
			}
		})
	}
}

func TestFactoryCachesByModel(t *testing.T) {
	t.Parallel()

	factory := gollemclient.NewFactory(
		vault{gollemclient.SecretRef(gollemclient.ProviderOpenAI): "key"},
		nil,
		newValues(t, nil),
	)

	first, err := factory.New(t.Context(), openaiRef())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	second, err := factory.New(t.Context(), openaiRef())
	if err != nil {
		t.Fatalf("New again: %v", err)
	}
	if first != second {
		t.Error("the factory built a second client for the same model")
	}

	other, err := factory.New(t.Context(), llm.ModelRef{Provider: gollemclient.ProviderOpenAI, Model: "gpt-5.6-luna"})
	if err != nil {
		t.Fatalf("New other: %v", err)
	}
	if first == other {
		t.Error("the factory reused a client across models")
	}
}

func TestSettings(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		base    string
		wantErr bool
	}{
		{name: "no override"},
		{name: "an https endpoint", base: "https://proxy.example/v1"},
		{name: "an http endpoint", base: "http://127.0.0.1:8080"},
		{name: "a hostless value", base: "not a url", wantErr: true},
		{name: "an unsupported scheme", base: "ftp://proxy.example", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(tc.base)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			values := settings.Default().NewValues()
			_, err = settings.Default().Apply(values, map[string]json.RawMessage{"llm.openai.baseUrl": encoded})
			if tc.wantErr != (err != nil) {
				t.Fatalf("Apply = %v, want error %t", err, tc.wantErr)
			}
			if !tc.wantErr && gollemclient.Timeout(values) != gollemclient.DefaultTimeout {
				t.Errorf("Timeout = %s, want the default", gollemclient.Timeout(values))
			}
		})
	}
}

func TestCompleteHonoursTheTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	values := newValues(t, map[string]string{"llm.openai.baseUrl": server.URL})
	factory := gollemclient.NewFactory(vault{gollemclient.SecretRef(gollemclient.ProviderOpenAI): "key"}, nil, values)
	client := gollemclient.New(factory, nil, 20*time.Millisecond)

	_, err := client.Complete(t.Context(), port.Request{Ref: openaiRef(), Messages: []port.Message{{Role: port.RoleUser, Text: "write"}}})
	if !errors.IsCode(err, errors.External) {
		t.Fatalf("Complete error = %v (%s), want %s", err, errors.CodeOf(err), errors.External)
	}
}

func TestStreamReportsAFailedStart(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, gollemclient.ProviderOpenAI, "llm.openai.baseUrl", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		writeJSON(t, w, `{"error":{"message":"down","type":"server_error"}}`)
	})

	_, err := client.Stream(t.Context(), port.Request{Ref: openaiRef(), Messages: []port.Message{{Role: port.RoleUser, Text: "write"}}})
	if !errors.IsCode(err, errors.External) {
		t.Fatalf("Stream error = %v (%s), want %s", err, errors.CodeOf(err), errors.External)
	}
}

func TestStreamStopsWhenTheCallerLeaves(t *testing.T) {
	t.Parallel()

	client, _ := newClient(t, gollemclient.ProviderOpenAI, "llm.openai.baseUrl", func(w http.ResponseWriter, _ *http.Request) {
		writeSSE(t, w, openaiStream)
	})

	ctx, cancel := context.WithCancel(t.Context())
	deltas, err := client.Stream(ctx, port.Request{Ref: openaiRef(), Messages: []port.Message{{Role: port.RoleUser, Text: "write"}}})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	cancel()
	for range deltas {
		continue
	}
}

func TestTemperatureReachesTheProvider(t *testing.T) {
	t.Parallel()

	warm := 0.2
	client, captured := newClient(t, gollemclient.ProviderOpenAI, "llm.openai.baseUrl", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, openaiCompletion)
	})

	if _, err := client.Complete(t.Context(), port.Request{
		Ref:         llm.ModelRef{Provider: gollemclient.ProviderOpenAI, Model: "gpt-4.1"},
		Messages:    []port.Message{{Role: port.RoleUser, Text: "write"}},
		Temperature: &warm,
	}); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !strings.Contains(captured.last(), `"temperature":0.2`) {
		t.Errorf("the request body carries no temperature: %s", captured.last())
	}
}

func TestClientWithoutATimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, openaiCompletion)
	}))
	t.Cleanup(server.Close)

	values := newValues(t, map[string]string{"llm.openai.baseUrl": server.URL})
	factory := gollemclient.NewFactory(vault{gollemclient.SecretRef(gollemclient.ProviderOpenAI): "key"}, nil, values)
	client := gollemclient.New(factory, nil, 0)

	resp, err := client.Complete(t.Context(), port.Request{Ref: openaiRef(), Messages: []port.Message{{Role: port.RoleUser, Text: "write"}}})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Text == "" {
		t.Error("the answer is empty")
	}
}

func TestGeminiThroughTheOpenAICompatibleEndpoint(t *testing.T) {
	t.Parallel()

	client, captured := newClient(t, gollemclient.ProviderGeminiOpenAI, "llm.geminiOpenai.baseUrl",
		func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, openaiCompletion)
		})

	resp, err := client.Complete(t.Context(), port.Request{
		Ref:       llm.ModelRef{Provider: gollemclient.ProviderGeminiOpenAI, Model: "gemini-3.5-flash"},
		System:    "you write pages",
		Messages:  []port.Message{{Role: port.RoleUser, Text: "write"}},
		MaxTokens: 256,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Text != "Koffein und Powder" {
		t.Fatalf("text = %q, want the provider answer", resp.Text)
	}
	if !strings.Contains(captured.last(), `"gemini-3.5-flash"`) {
		t.Fatalf("the request body is %s, want the gemini model", captured.last())
	}
}

func TestGeminiOpenAIDefaultsToGoogleAI(t *testing.T) {
	t.Parallel()

	if !strings.HasPrefix(gollemclient.DefaultGeminiOpenAIBaseURL, "https://generativelanguage.googleapis.com/") {
		t.Fatalf("the default base URL is %q", gollemclient.DefaultGeminiOpenAIBaseURL)
	}

	values := settings.Default().NewValues()
	factory := gollemclient.NewFactory(vault{}, nil, values)
	if _, err := factory.New(t.Context(), llm.ModelRef{
		Provider: gollemclient.ProviderGeminiOpenAI, Model: "gemini-3.5-flash",
	}); !errors.IsCode(err, errors.Unauthorized) {
		t.Fatalf("New without a key = %v, want %s", err, errors.Unauthorized)
	}
}

func TestCompleteCarriesAnOutputBudgetAReasoningModelCanAnswerWithin(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		info  llm.ModelInfo
		asked int
		want  string
	}{
		{
			name:  "a reasoning model is given room to think and to answer",
			info:  llm.ModelInfo{Ref: openaiRef(), Reasoning: true, ReasoningEffort: llm.EffortLow, MaxOutputTokens: 128000},
			asked: 16,
			want:  `"max_completion_tokens":2064`,
		},
		{
			name:  "a model that does not reason is asked for exactly what the caller wanted",
			info:  llm.ModelInfo{Ref: openaiRef(), MaxOutputTokens: 128000},
			asked: 16,
			want:  `"max_completion_tokens":16`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client, captured := newClientOver(t, gollemclient.ProviderOpenAI, "llm.openai.baseUrl",
				catalog{openaiRef().String(): tc.info},
				func(w http.ResponseWriter, _ *http.Request) {
					writeJSON(t, w, openaiCompletion)
				})

			resp, err := client.Complete(t.Context(), port.Request{
				Ref:       openaiRef(),
				Messages:  []port.Message{{Role: port.RoleUser, Text: "ping"}},
				MaxTokens: tc.asked,
			})
			if err != nil {
				t.Fatalf("Complete: %v", err)
			}
			if resp.FinishReason != port.FinishStop {
				t.Errorf("finish reason = %s, want %s", resp.FinishReason, port.FinishStop)
			}
			if sent := captured.last(); !strings.Contains(sent, tc.want) {
				t.Errorf("the request body does not carry %s: %s", tc.want, sent)
			}
		})
	}
}

func TestCompleteFallsBackToTheAskedCeilingWhenTheCatalogHasNoRow(t *testing.T) {
	t.Parallel()

	client, captured := newClientOver(t, gollemclient.ProviderOpenAI, "llm.openai.baseUrl", catalog{},
		func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, openaiCompletion)
		})

	if _, err := client.Complete(t.Context(), port.Request{
		Ref:       openaiRef(),
		Messages:  []port.Message{{Role: port.RoleUser, Text: "ping"}},
		MaxTokens: 128,
	}); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if sent := captured.last(); !strings.Contains(sent, `"max_completion_tokens":128`) {
		t.Errorf("the request body does not carry the asked ceiling: %s", sent)
	}
}
