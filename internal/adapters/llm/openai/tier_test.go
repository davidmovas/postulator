package openai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/openai"
	"github.com/davidmovas/postulator/internal/adapters/llm/openai/openaitest"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func onFlex(text string) port.Request {
	req := write(text)
	req.Tier = llm.TierFlex
	return req
}

func tiers(t *testing.T, server *openaitest.Server) []any {
	t.Helper()

	requests := server.Requests()
	sent := make([]any, 0, len(requests))
	for _, request := range requests {
		sent = append(sent, request.Body["service_tier"])
	}
	return sent
}

func canonicalJSON(t *testing.T, value any) string {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode %v: %v", value, err)
	}
	return string(encoded)
}

func sameTiers(got []any, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestAFlexCompletionFallsBackToTheDefaultTierOnce(t *testing.T) {
	t.Parallel()

	served := openaitest.Answer{Text: "on default", Usage: openaitest.Usage{Input: 100, Output: 20}}.Reply()
	cases := []struct {
		name     string
		replies  []openaitest.Reply
		patience time.Duration
		want     errors.Code
		text     string
		tiers    []string
	}{
		{
			name:    "flex had no capacity",
			replies: []openaitest.Reply{openaitest.FlexCapacity(), served},
			text:    "on default", tiers: []string{"flex", "default"},
		},
		{
			name:     "flex did not answer within the patience",
			replies:  []openaitest.Reply{openaitest.Text("too late").Reply().After(5 * time.Second), served},
			patience: 100 * time.Millisecond,
			text:     "on default", tiers: []string{"flex", "default"},
		},
		{
			name:     "flex answered in time",
			replies:  []openaitest.Reply{openaitest.Answer{Text: "on flex", Tier: "flex"}.Reply().After(20 * time.Millisecond)},
			patience: 2 * time.Second,
			text:     "on flex", tiers: []string{"flex"},
		},
		{
			name:    "flex without patience waits",
			replies: []openaitest.Reply{openaitest.Answer{Text: "on flex", Tier: "flex"}.Reply().After(150 * time.Millisecond)},
			text:    "on flex", tiers: []string{"flex"},
		},
		{
			name:    "the credit is gone on flex too",
			replies: []openaitest.Reply{openaitest.QuotaExhausted(), served},
			want:    errors.NeedsHuman, tiers: []string{"flex"},
		},
		{
			name:    "an ordinary rate limit is no reason to leave flex",
			replies: []openaitest.Reply{openaitest.Failure(http.StatusTooManyRequests, openaitest.RateLimit()), served},
			want:    errors.RateLimited, tiers: []string{"flex"},
		},
		{
			name:    "a server error is the retry's to handle",
			replies: []openaitest.Reply{openaitest.Failure(http.StatusInternalServerError, openaitest.ServerError()), served},
			want:    errors.External, tiers: []string{"flex"},
		},
		{
			name:    "the default tier fails as well",
			replies: []openaitest.Reply{openaitest.FlexCapacity(), openaitest.Failure(http.StatusServiceUnavailable, openaitest.ServerError())},
			want:    errors.External, tiers: []string{"flex", "default"},
		},
		{
			name:    "the default tier has no room either",
			replies: []openaitest.Reply{openaitest.FlexCapacity(), openaitest.FlexCapacity()},
			want:    errors.RateLimited, tiers: []string{"flex", "default"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			server.Enqueue(tc.replies...)

			resp, err := newClient(server, openai.WithFlexPatience(tc.patience)).Complete(t.Context(), onFlex("hello"))
			if tc.want != "" {
				if !errors.IsCode(err, tc.want) {
					t.Fatalf("Complete = %v (%s), want %s", err, errors.CodeOf(err), tc.want)
				}
			} else if err != nil {
				t.Fatalf("Complete: %v", err)
			}
			if resp.Text != tc.text {
				t.Errorf("text = %q, want %q", resp.Text, tc.text)
			}
			if got := tiers(t, server); !sameTiers(got, tc.tiers...) {
				t.Errorf("tiers sent = %v, want %v", got, tc.tiers)
			}
		})
	}
}

func TestAResendReportsTheAttemptThatAnswered(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.FlexCapacity(), openaitest.Answer{
		Text: "on default", Usage: openaitest.Usage{Input: 1200, Cached: 1024, Output: 300, Reasoning: 120},
	}.Reply())

	resp, err := newClient(server).Complete(t.Context(), onFlex("hello"))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Tier != llm.TierDefault {
		t.Errorf("tier = %s, want the default that served the resend", resp.Tier)
	}
	if (resp.Usage != llm.Usage{Input: 1200, CachedInput: 1024, Output: 300, Reasoning: 120, Total: 1500}) {
		t.Errorf("usage = %+v, want the resend's own", resp.Usage)
	}

	requests := server.Requests()
	if len(requests) != 2 {
		t.Fatalf("the server saw %d requests, want the flex attempt and the resend", len(requests))
	}
	first, second := requests[0].Body, requests[1].Body
	for _, key := range []string{"model", "input", "max_output_tokens", "reasoning", "prompt_cache_options"} {
		if canonicalJSON(t, first[key]) != canonicalJSON(t, second[key]) {
			t.Errorf("the resend changed %s: %v, then %v", key, first[key], second[key])
		}
	}
}

func TestPatienceIsOnlyForFlex(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("slow but default").Reply().After(250 * time.Millisecond))

	resp, err := newClient(server, openai.WithFlexPatience(50*time.Millisecond)).Complete(t.Context(), write("hello"))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Text != "slow but default" || len(server.Requests()) != 1 {
		t.Errorf("text = %q after %d requests, want the one default answer", resp.Text, len(server.Requests()))
	}
}

func TestACallerWhoLeavesDuringFlexIsNotResent(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("late").Reply().After(5*time.Second), openaitest.Text("never").Reply())

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := newClient(server, openai.WithFlexPatience(time.Second)).Complete(ctx, onFlex("hello"))
	if !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("Complete = %v (%s), want %s", err, errors.CodeOf(err), errors.Cancelled)
	}
	if got := tiers(t, server); !sameTiers(got, "flex") {
		t.Errorf("tiers sent = %v, want only the flex attempt", got)
	}
}

func TestAClientTimeoutDuringFlexIsNotResent(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("late").Reply().After(5*time.Second), openaitest.Text("never").Reply())

	_, err := newClient(server, openai.WithTimeout(60*time.Millisecond), openai.WithFlexPatience(time.Second)).Complete(t.Context(), onFlex("hello"))
	if !errors.IsCode(err, errors.External) {
		t.Fatalf("Complete = %v (%s), want %s", err, errors.CodeOf(err), errors.External)
	}
	if got := tiers(t, server); !sameTiers(got, "flex") {
		t.Errorf("tiers sent = %v, want only the flex attempt", got)
	}
}

func TestAFlexStreamFallsBackBeforeItHasSpoken(t *testing.T) {
	t.Parallel()

	stalled := openaitest.Text("too late").Stream()
	stalled.Events[2].Pause = 5 * time.Second

	served := openaitest.Answer{Text: "on default", Chunks: []string{"on", " default"}}.Stream()
	cases := []struct {
		name     string
		replies  []openaitest.Reply
		patience time.Duration
		tiers    []string
	}{
		{name: "no capacity over http", replies: []openaitest.Reply{openaitest.FlexCapacity(), served}, tiers: []string{"flex", "default"}},
		{name: "no capacity inside the stream", replies: []openaitest.Reply{openaitest.StreamFailure(openaitest.Capacity()), served}, tiers: []string{"flex", "default"}},
		{
			name:    "no answer within the patience",
			replies: []openaitest.Reply{openaitest.Text("too late").Stream().After(5 * time.Second), served}, patience: 100 * time.Millisecond,
			tiers: []string{"flex", "default"},
		},
		{
			name:    "admitted but silent past the patience",
			replies: []openaitest.Reply{stalled, served}, patience: 100 * time.Millisecond,
			tiers: []string{"flex", "default"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			server.Enqueue(tc.replies...)

			deltas, err := newClient(server, openai.WithFlexPatience(tc.patience)).Stream(t.Context(), onFlex("hello"))
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}
			got := listen(t, deltas)
			if got.text() != "on default" || got.done == nil || got.done.Tier != llm.TierDefault {
				t.Errorf("stream = %+v (final %+v), want the default tier's answer", got, got.done)
			}
			if sent := tiers(t, server); !sameTiers(sent, tc.tiers...) {
				t.Errorf("tiers sent = %v, want %v", sent, tc.tiers)
			}
		})
	}
}

func TestAFlexStreamThatHasSpokenIsNotCutByThePatience(t *testing.T) {
	t.Parallel()

	slow := openaitest.Answer{Text: "first second", Chunks: []string{"first", " second"}, Tier: "flex"}.Stream()
	for i := range slow.Events {
		if slow.Events[i].Name == "response.output_text.done" {
			slow.Events[i].Pause = 300 * time.Millisecond
		}
	}

	server := openaitest.New(t)
	server.Enqueue(slow)

	deltas, err := newClient(server, openai.WithFlexPatience(100*time.Millisecond)).Stream(t.Context(), onFlex("hello"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	got := listen(t, deltas)
	if got.err != nil || got.text() != "first second" || got.done == nil || got.done.Tier != llm.TierFlex {
		t.Errorf("stream = %+v (final %+v), want the whole flex answer", got, got.done)
	}
	if len(server.Requests()) != 1 {
		t.Errorf("the server saw %d requests, want one", len(server.Requests()))
	}
}

func TestAFlexStreamRefusedForCreditIsNotResent(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.StreamFailure(openaitest.Quota()), openaitest.Text("never").Stream())

	_, err := newClient(server).Stream(t.Context(), onFlex("hello"))
	if !errors.IsCode(err, errors.NeedsHuman) {
		t.Fatalf("Stream = %v (%s), want %s", err, errors.CodeOf(err), errors.NeedsHuman)
	}
	if got := tiers(t, server); !sameTiers(got, "flex") {
		t.Errorf("tiers sent = %v, want only the flex attempt", got)
	}
}

func TestAFlexStreamWhoseResendFailsSaysWhy(t *testing.T) {
	t.Parallel()

	server := openaitest.New(t)
	server.Enqueue(openaitest.FlexCapacity(), openaitest.QuotaExhausted())

	_, err := newClient(server).Stream(t.Context(), onFlex("hello"))
	if !errors.IsCode(err, errors.NeedsHuman) {
		t.Fatalf("Stream = %v (%s), want %s", err, errors.CodeOf(err), errors.NeedsHuman)
	}
	if got := tiers(t, server); !sameTiers(got, "flex", "default") {
		t.Errorf("tiers sent = %v, want the flex attempt and the resend", got)
	}
}
