package fake_test

import (
	"context"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func request(prompt string) port.Request {
	return port.Request{
		Ref:      llm.ModelRef{Provider: "openai", Model: "gpt-5.6-luna"},
		System:   "you write pages",
		Messages: []port.Message{{Role: port.RoleUser, Text: prompt}},
	}
}

func TestDirectives(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		prompt     string
		wantText   string
		wantReason port.FinishReason
		wantErr    errors.Code
	}{
		{name: "no directive", prompt: "write a page", wantText: "ok", wantReason: port.FinishStop},
		{name: "an answer", prompt: "ANSWER: Koffein und Powder", wantText: "Koffein und Powder", wantReason: port.FinishStop},
		{name: "json", prompt: `JSON: {"title":"Koffein"}`, wantText: `{"title":"Koffein"}`, wantReason: port.FinishStop},
		{name: "a truncated answer", prompt: "ANSWER: half a\nLENGTH", wantText: "half a", wantReason: port.FinishLength},
		{name: "a filtered answer", prompt: "FILTERED", wantReason: port.FinishContentFilter},
		{name: "a rate limit", prompt: "ERROR: RATE_LIMITED", wantErr: errors.RateLimited},
		{name: "an external failure", prompt: "write\nERROR: EXTERNAL", wantErr: errors.External},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := fake.New()
			resp, err := client.Complete(t.Context(), request(tc.prompt))
			if tc.wantErr != "" {
				if !errors.IsCode(err, tc.wantErr) {
					t.Fatalf("Complete error = %v, want %s", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Complete: %v", err)
			}
			if resp.Text != tc.wantText {
				t.Errorf("text = %q, want %q", resp.Text, tc.wantText)
			}
			if resp.FinishReason != tc.wantReason {
				t.Errorf("finish reason = %s, want %s", resp.FinishReason, tc.wantReason)
			}
			if resp.Usage.Total != resp.Usage.Input+resp.Usage.Output {
				t.Errorf("usage = %+v, want a consistent total", resp.Usage)
			}
			if len(client.Requests()) != 1 {
				t.Errorf("requests = %d, want 1", len(client.Requests()))
			}
		})
	}
}

func TestStream(t *testing.T) {
	t.Parallel()

	client := fake.New()
	deltas, err := client.Stream(t.Context(), request("ANSWER: Koffein und Powder"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	var (
		text  strings.Builder
		usage llm.Usage
		done  bool
	)
	for delta := range deltas {
		text.WriteString(delta.Text)
		if delta.Done {
			done = true
			usage = *delta.Usage
		}
	}

	if !done || text.String() != "Koffein und Powder" {
		t.Fatalf("stream = %q, done %t, want the scripted answer", text.String(), done)
	}
	if usage.Output == 0 {
		t.Errorf("usage = %+v, want output tokens", usage)
	}
}

func TestStreamRefusals(t *testing.T) {
	t.Parallel()

	client := fake.New()
	if _, err := client.Stream(t.Context(), request("ERROR: UNAUTHORIZED")); !errors.IsCode(err, errors.Unauthorized) {
		t.Fatalf("Stream error = %v, want %s", err, errors.Unauthorized)
	}

	deltas, err := client.Stream(t.Context(), request("FILTERED"))
	if err != nil {
		t.Fatalf("Stream filtered: %v", err)
	}
	for delta := range deltas {
		if delta.Text != "" {
			t.Errorf("filtered stream produced %q", delta.Text)
		}
	}
}

func TestInvalidRequestsAreRefused(t *testing.T) {
	t.Parallel()

	client := fake.New()
	if _, err := client.Complete(t.Context(), port.Request{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Complete error = %v, want %s", err, errors.Invalid)
	}
	if _, err := client.Stream(t.Context(), port.Request{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Stream error = %v, want %s", err, errors.Invalid)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := client.Complete(ctx, request("write")); !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("Complete error = %v, want %s", err, errors.Cancelled)
	}
	if _, err := client.Stream(ctx, request("write")); !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("Stream error = %v, want %s", err, errors.Cancelled)
	}
}
