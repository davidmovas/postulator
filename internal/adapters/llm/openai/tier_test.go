package openai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
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

const (
	held           = time.Minute
	patienceAsked  = 8 * time.Minute
	alarmPending   = "pending"
	alarmStopped   = "stopped"
	alarmRungState = "rung"
)

type patience struct {
	alarms []*fakeAlarm
	mu     sync.Mutex
}

type fakeAlarm struct {
	owner *patience
	ring  func()
	state string
	after time.Duration
}

func (p *patience) arm(after time.Duration, ring func()) openai.Alarm {
	p.mu.Lock()
	defer p.mu.Unlock()

	armed := &fakeAlarm{owner: p, ring: ring, state: alarmPending, after: after}
	p.alarms = append(p.alarms, armed)
	return armed
}

func (a *fakeAlarm) Stop() bool {
	a.owner.mu.Lock()
	defer a.owner.mu.Unlock()

	if a.state != alarmPending {
		return false
	}
	a.state = alarmStopped
	return true
}

func (p *patience) ringAll() {
	p.mu.Lock()
	due := make([]func(), 0, len(p.alarms))
	for _, armed := range p.alarms {
		if armed.state == alarmPending {
			armed.state = alarmRungState
			due = append(due, armed.ring)
		}
	}
	p.mu.Unlock()

	for _, ring := range due {
		ring()
	}
}

func (p *patience) armed() []time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]time.Duration, 0, len(p.alarms))
	for _, armed := range p.alarms {
		out = append(out, armed.after)
	}
	return out
}

func (p *patience) rung() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	count := 0
	for _, armed := range p.alarms {
		if armed.state == alarmRungState {
			count++
		}
	}
	return count
}

func patientClient(server *openaitest.Server, after time.Duration, alarms *patience, opts ...openai.Option) *openai.Client {
	return newClient(server, append([]openai.Option{openai.WithFlexPatience(after), openai.WithAlarm(alarms.arm)}, opts...)...)
}

func stalled(reply openaitest.Reply, alarms *patience) openaitest.Reply {
	return reply.After(held).OnArrival(alarms.ringAll)
}

func sameDurations(got []time.Duration, want ...time.Duration) bool {
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
	late := &patience{}
	cases := []struct {
		name     string
		alarms   *patience
		replies  []openaitest.Reply
		patience time.Duration
		want     errors.Code
		text     string
		tiers    []string
		armed    []time.Duration
	}{
		{
			name:    "flex had no capacity",
			replies: []openaitest.Reply{openaitest.FlexCapacity(), served},
			text:    "on default", tiers: []string{"flex", "default"},
		},
		{
			name:     "flex did not answer within the patience",
			alarms:   late,
			replies:  []openaitest.Reply{stalled(openaitest.Text("too late").Reply(), late), served},
			patience: patienceAsked,
			text:     "on default", tiers: []string{"flex", "default"}, armed: []time.Duration{patienceAsked},
		},
		{
			name:     "flex answered in time",
			replies:  []openaitest.Reply{openaitest.Answer{Text: "on flex", Tier: "flex"}.Reply()},
			patience: patienceAsked,
			text:     "on flex", tiers: []string{"flex"}, armed: []time.Duration{patienceAsked},
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
			alarms := tc.alarms
			if alarms == nil {
				alarms = &patience{}
			}

			resp, err := patientClient(server, tc.patience, alarms).Complete(t.Context(), onFlex("hello"))
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
			if got := alarms.armed(); !sameDurations(got, tc.armed...) {
				t.Errorf("patience armed = %v, want %v: only a flex attempt with a patience waits on one", got, tc.armed)
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
	server.Enqueue(openaitest.Text("slow but default").Reply().After(50 * time.Millisecond))

	alarms := &patience{}
	resp, err := patientClient(server, patienceAsked, alarms).Complete(t.Context(), write("hello"))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Text != "slow but default" || len(server.Requests()) != 1 {
		t.Errorf("text = %q after %d requests, want the one default answer", resp.Text, len(server.Requests()))
	}
	if armed := alarms.armed(); len(armed) != 0 {
		t.Errorf("a default call armed the patience %v", armed)
	}
}

func TestACallerWhoLeavesDuringFlexIsNotResent(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	server := openaitest.New(t)
	server.Enqueue(openaitest.Text("late").Reply().After(held).OnArrival(cancel), openaitest.Text("never").Reply())

	_, err := patientClient(server, patienceAsked, &patience{}).Complete(ctx, onFlex("hello"))
	if !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("Complete = %v (%s), want %s", err, errors.CodeOf(err), errors.Cancelled)
	}
	if got := tiers(t, server); !sameTiers(got, "flex") {
		t.Errorf("tiers sent = %v, want only the flex attempt", got)
	}
}

type silentProvider struct {
	tiers []any
	mu    sync.Mutex
}

func (s *silentProvider) RoundTrip(req *http.Request) (*http.Response, error) {
	var body map[string]any
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.tiers = append(s.tiers, body["service_tier"])
	s.mu.Unlock()

	<-req.Context().Done()
	return nil, req.Context().Err()
}

func (s *silentProvider) sent() []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]any(nil), s.tiers...)
}

func TestAClientTimeoutDuringFlexIsNotResent(t *testing.T) {
	t.Parallel()

	silent := &silentProvider{}
	client := openai.New(keys(), models(),
		openai.WithBaseURL("http://provider.invalid/v1"),
		openai.WithHTTPClient(&http.Client{Transport: silent}),
		openai.WithTimeout(60*time.Millisecond),
		openai.WithFlexPatience(patienceAsked), openai.WithAlarm((&patience{}).arm))

	_, err := client.Complete(t.Context(), onFlex("hello"))
	if !errors.IsCode(err, errors.External) {
		t.Fatalf("Complete = %v (%s), want %s", err, errors.CodeOf(err), errors.External)
	}
	if got := silent.sent(); !sameTiers(got, "flex") {
		t.Errorf("tiers sent = %v, want only the flex attempt", got)
	}
}

func TestAFlexStreamFallsBackBeforeItHasSpoken(t *testing.T) {
	t.Parallel()

	served := openaitest.Answer{Text: "on default", Chunks: []string{"on", " default"}}.Stream()
	unanswered, silent := &patience{}, &patience{}
	admitted := openaitest.Text("too late").Stream()
	admitted.Events[2].Before = silent.ringAll
	admitted.Events[2].Pause = held

	cases := []struct {
		name     string
		alarms   *patience
		replies  []openaitest.Reply
		patience time.Duration
		tiers    []string
		rung     int
	}{
		{name: "no capacity over http", replies: []openaitest.Reply{openaitest.FlexCapacity(), served}, tiers: []string{"flex", "default"}},
		{name: "no capacity inside the stream", replies: []openaitest.Reply{openaitest.StreamFailure(openaitest.Capacity()), served}, tiers: []string{"flex", "default"}},
		{
			name: "no answer within the patience", alarms: unanswered, patience: patienceAsked,
			replies: []openaitest.Reply{stalled(openaitest.Text("too late").Stream(), unanswered), served},
			tiers:   []string{"flex", "default"}, rung: 1,
		},
		{
			name: "admitted but silent past the patience", alarms: silent, patience: patienceAsked,
			replies: []openaitest.Reply{admitted, served},
			tiers:   []string{"flex", "default"}, rung: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := openaitest.New(t)
			server.Enqueue(tc.replies...)
			alarms := tc.alarms
			if alarms == nil {
				alarms = &patience{}
			}

			deltas, err := patientClient(server, tc.patience, alarms).Stream(t.Context(), onFlex("hello"))
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
			if rung := alarms.rung(); rung != tc.rung {
				t.Errorf("the patience rang %d times, want %d", rung, tc.rung)
			}
		})
	}
}

func TestAFlexStreamThatHasSpokenIsNotCutByThePatience(t *testing.T) {
	t.Parallel()

	gate := make(chan struct{})
	release := sync.OnceFunc(func() { close(gate) })
	slow := openaitest.Answer{Text: "first second", Chunks: []string{"first", " second"}, Tier: "flex"}.Stream()
	for i := range slow.Events {
		if slow.Events[i].Name == "response.output_text.delta" {
			slow.Events[i].Before = func() { <-gate }
			break
		}
	}

	server := openaitest.New(t)
	t.Cleanup(release)
	server.Enqueue(slow)

	alarms := &patience{}
	deltas, err := patientClient(server, patienceAsked, alarms).Stream(t.Context(), onFlex("hello"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	alarms.ringAll()
	release()

	got := listen(t, deltas)
	if got.err != nil || got.text() != "first second" || got.done == nil || got.done.Tier != llm.TierFlex {
		t.Errorf("stream = %+v (final %+v), want the whole flex answer", got, got.done)
	}
	if len(server.Requests()) != 1 {
		t.Errorf("the server saw %d requests, want one", len(server.Requests()))
	}
	if armed := alarms.armed(); !sameDurations(armed, patienceAsked) || alarms.rung() != 0 {
		t.Errorf("patience armed %v and rang %d times, want one stopped before it could ring", armed, alarms.rung())
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
