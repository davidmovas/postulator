package agent

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type quietStream struct {
	deltas []string
	rounds []agentapp.RoundUsage
	waits  []agentapp.Wait
}

func (q *quietStream) Delta(_ context.Context, _ int64, text string) error {
	q.deltas = append(q.deltas, text)
	return nil
}

func (q *quietStream) Spent(_ context.Context, round agentapp.RoundUsage) error {
	q.rounds = append(q.rounds, round)
	return nil
}

func (q *quietStream) Waiting(_ context.Context, held agentapp.Wait) error {
	q.waits = append(q.waits, held)
	return nil
}

func (q *quietStream) ToolStarted(context.Context, string, string, json.RawMessage) error {
	return nil
}

func (q *quietStream) ToolFinished(context.Context, agentapp.ToolOutcome) error {
	return nil
}

type scriptedStream struct {
	rounds [][]llm.Delta
	asked  []llm.Request
	err    error
}

func (s *scriptedStream) Stream(_ context.Context, req llm.Request) (<-chan llm.Delta, error) {
	s.asked = append(s.asked, req)
	if s.err != nil {
		return nil, s.err
	}
	round := s.rounds[0]
	s.rounds = s.rounds[1:]

	out := make(chan llm.Delta, len(round))
	for _, delta := range round {
		out <- delta
	}
	close(out)
	return out, nil
}

type pricedCatalog struct{}

func (pricedCatalog) Lookup(context.Context, domainllm.ModelRef) (domainllm.ModelInfo, error) {
	return domainllm.ModelInfo{
		InputUSDPerM: 2, CachedInputUSDPerM: 0.2, OutputUSDPerM: 12,
		FlexInputUSDPerM: 1, FlexCachedInputUSDPerM: 0.1, FlexOutputUSDPerM: 6,
	}, nil
}

func text(pieces ...string) []llm.Delta {
	out := make([]llm.Delta, 0, len(pieces))
	for _, piece := range pieces {
		out = append(out, llm.Delta{Text: piece})
	}
	return out
}

func call(id string) llm.Delta {
	return llm.Delta{Call: &llm.ToolCall{ID: id, Name: "entities_count", Args: json.RawMessage(`{}`)}}
}

func done(input, cached, output int, tier domainllm.ServiceTier) llm.Delta {
	used := domainllm.Usage{Input: input, CachedInput: cached, Output: output, Total: input + output}
	return llm.Delta{Done: true, Usage: &used, Finish: llm.FinishStop, Tier: tier}
}

func round(parts ...[]llm.Delta) []llm.Delta {
	var out []llm.Delta
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}

func one(delta llm.Delta) []llm.Delta {
	return []llm.Delta{delta}
}

func searchedFor(kind llm.SearchKind, payload string) llm.Delta {
	return llm.Delta{Search: &llm.ToolSearch{Kind: kind, Execution: "server", Payload: json.RawMessage(payload)}}
}

func namespacedCall(id string) llm.Delta {
	return llm.Delta{Call: &llm.ToolCall{ID: id, Name: "pages_count", Args: json.RawMessage(`{}`), Namespace: "pages"}}
}

func played(t *testing.T, client streamer, stream agentapp.Stream, limit int) (*turn, error) {
	t.Helper()
	return playedWith(t, client, stream, limit, func(*Deps) {})
}

func playedWith(t *testing.T, client streamer, stream agentapp.Stream, limit int, adjust func(*Deps)) (*turn, error) {
	t.Helper()

	deps := Deps{
		Client:   client,
		Registry: tools.New(tools.Deps{Clock: clock.NewFake(time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC))}),
		Catalog:  pricedCatalog{},
		Clock:    clock.NewFake(time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)),
		Logger:   zaptest.NewLogger(t),
	}
	adjust(&deps)
	runner := New(deps)
	current := runner.open(agentapp.RunSpec{
		Binding:   tools.Binding{ConversationID: "c1", Mode: domainagent.ModeAutonomous},
		Ref:       domainllm.ModelRef{Provider: "openai", Model: "gpt-5.6-terra"},
		Input:     "how many entities?",
		Stream:    stream,
		LoopLimit: limit,
	}, "the instructions", "the site")
	return current, runner.execute(t.Context(), current)
}

func TestEveryRoundIsCountedOnceAndBilledAtTheTierItWasServed(t *testing.T) {
	t.Parallel()

	client := &scriptedStream{rounds: [][]llm.Delta{
		round(one(call("c1")), one(done(1200, 0, 40, domainllm.TierDefault))),
		round(one(call("c2")), one(done(1800, 1152, 30, domainllm.TierFlex))),
		round(text("Twelve ", "entities."), one(done(2400, 2048, 12, ""))),
	}}
	stream := &quietStream{}

	current, err := played(t, client, stream, 4)
	if err != nil {
		t.Fatalf("the turn failed: %v", err)
	}

	if len(stream.rounds) != 3 {
		t.Fatalf("the window was told about %d rounds, want one per model call", len(stream.rounds))
	}
	for index, spent := range stream.rounds {
		if spent.Round != index+1 || spent.Provider != "openai" || spent.Model != "gpt-5.6-terra" {
			t.Fatalf("round %d reads %+v", index+1, spent)
		}
	}

	want := []float64{
		(1200*2 + 40*12) / 1e6,
		(648*1 + 1152*0.1 + 30*6) / 1e6,
		(352*2 + 2048*0.2 + 12*12) / 1e6,
	}
	total := 0.0
	for index, spent := range stream.rounds {
		if math.Abs(spent.USD-want[index]) > 1e-12 {
			t.Fatalf("round %d cost %v, want %v", index+1, spent.USD, want[index])
		}
		total += want[index]
	}

	if got := current.usage.total(); got.Input != 5400 || got.CachedInput != 3200 || got.Output != 82 {
		t.Fatalf("the turn counted %+v", got)
	}
	if current.usage.answered() != 3 || math.Abs(current.usage.spent()-total) > 1e-12 {
		t.Fatalf("the turn counted %d calls costing %v", current.usage.answered(), current.usage.spent())
	}
}

func TestTheAnswerIsTheTextTheTurnStreamed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		rounds [][]llm.Delta
		want   string
	}{
		{
			name:   "the pieces of one round are joined with nothing",
			rounds: [][]llm.Delta{round(text("От", "лич", "ная", " идея", "."), one(done(10, 0, 5, "")))},
			want:   "Отличная идея.",
		},
		{
			name: "a round that only called a tool says nothing",
			rounds: [][]llm.Delta{
				round(one(call("c1")), one(done(10, 0, 5, ""))),
				round(text("Twelve entities."), one(done(20, 0, 3, ""))),
			},
			want: "Twelve entities.",
		},
		{
			name: "two speaking rounds are separated by a blank line",
			rounds: [][]llm.Delta{
				round(text("Let me look."), one(call("c1")), one(done(10, 0, 5, ""))),
				round(text("Twelve entities."), one(done(20, 0, 3, ""))),
			},
			want: "Let me look.\n\nTwelve entities.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stream := &quietStream{}
			current, err := played(t, &scriptedStream{rounds: tc.rounds}, stream, 4)
			if err != nil {
				t.Fatalf("the turn failed: %v", err)
			}
			if got := current.spoken.String(); got != tc.want {
				t.Errorf("answer = %q, want %q", got, tc.want)
			}

			streamed, sent := "", ""
			for _, piece := range stream.deltas {
				streamed += piece
			}
			for _, deltas := range tc.rounds {
				for _, delta := range deltas {
					sent += delta.Text
				}
			}
			if streamed != sent {
				t.Errorf("the stream carried %q, want %q", streamed, sent)
			}
		})
	}
}

func TestARoundThatDidNotFinishFailsTheTurn(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		client *scriptedStream
		want   errors.Code
	}{
		{
			name:   "a refusal at admission keeps its code",
			client: &scriptedStream{err: errors.New(errors.NeedsHuman, "the account is out of credit")},
			want:   errors.NeedsHuman,
		},
		{
			name: "a failure in the stream keeps its code",
			client: &scriptedStream{rounds: [][]llm.Delta{
				round(text("Half"), one(llm.Delta{Err: errors.New(errors.RateLimited, "slow down")})),
			}},
			want: errors.RateLimited,
		},
		{
			name:   "a stream that ends without its end is the provider's fault",
			client: &scriptedStream{rounds: [][]llm.Delta{text("Half an answer")}},
			want:   errors.External,
		},
		{
			name: "a turn that keeps calling tools runs out of budget",
			client: &scriptedStream{rounds: [][]llm.Delta{
				round(one(call("c1")), one(done(10, 0, 5, ""))),
				round(one(call("c2")), one(done(10, 0, 5, ""))),
			}},
			want: errors.BudgetExceeded,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := played(t, tc.client, &quietStream{}, 2); !errors.IsCode(err, tc.want) {
				t.Fatalf("the turn ended with %v, want %s", err, tc.want)
			}
		})
	}
}

func TestEveryCallOfARoundIsAnsweredInOrder(t *testing.T) {
	t.Parallel()

	client := &scriptedStream{rounds: [][]llm.Delta{
		round(one(call("c1")), one(call("c2")), one(done(10, 0, 5, ""))),
		round(text("done"), one(done(20, 0, 3, ""))),
	}}

	if _, err := played(t, client, nil, 4); err != nil {
		t.Fatalf("the turn failed: %v", err)
	}

	second := client.asked[1].Messages
	tail := second[len(second)-4:]
	if tail[0].Call == nil || tail[0].Call.ID != "c1" || tail[1].Call == nil || tail[1].Call.ID != "c2" {
		t.Fatalf("the calls were replayed as %+v", tail)
	}
	if tail[2].Result == nil || tail[2].Result.CallID != "c1" || tail[3].Result == nil || tail[3].Result.CallID != "c2" {
		t.Fatalf("the results were sent as %+v", tail)
	}
	if !json.Valid(tail[2].Result.Output) {
		t.Fatalf("a result is not JSON: %s", tail[2].Result.Output)
	}
}

func TestATurnOffersItsToolsTheWayTheSettingLoadsThem(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		loading  func() agentapp.ToolLoading
		deferred bool
	}{
		{name: "no setting sends every tool every round"},
		{name: "every tool every round", loading: func() agentapp.ToolLoading { return agentapp.ToolsEveryRound }},
		{name: "tools loaded on demand", loading: func() agentapp.ToolLoading { return agentapp.ToolsOnDemand }, deferred: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := &scriptedStream{rounds: [][]llm.Delta{round(text("hi"), one(done(10, 0, 2, "")))}}
			if _, err := playedWith(t, client, nil, 2, func(d *Deps) { d.ToolLoading = tc.loading }); err != nil {
				t.Fatalf("the turn failed: %v", err)
			}

			whole, grouped := 0, 0
			for _, tool := range client.asked[0].Tools {
				group, onDemand := tools.OnDemand(tool.Name)
				switch {
				case tool.Deferred == nil:
					whole++
					if tc.deferred && onDemand {
						t.Errorf("%s was sent whole although it loads on demand", tool.Name)
					}
				case !tc.deferred:
					t.Errorf("%s was deferred while every tool goes every round", tool.Name)
				case tool.Deferred.Name != group.Name || tool.Deferred.Description != group.Description:
					t.Errorf("%s was offered in %+v, want its group %+v", tool.Name, tool.Deferred, group)
				default:
					grouped++
				}
			}
			if tc.deferred && (whole == 0 || grouped == 0) {
				t.Errorf("a deferred turn sent %d tools whole and %d in groups, want both", whole, grouped)
			}
		})
	}
}

func TestAToolSearchIsReplayedWithinItsTurnAndForgottenAfterIt(t *testing.T) {
	t.Parallel()

	client := &scriptedStream{rounds: [][]llm.Delta{
		round(one(searchedFor(llm.SearchCall, `{"paths":["pages"]}`)), one(searchedFor(llm.SearchOutput, `[]`)),
			one(namespacedCall("c1")), one(done(10, 0, 5, ""))),
		round(text("done"), one(done(20, 0, 3, ""))),
	}}
	stored := &heldHistory{}

	_, err := playedWith(t, client, nil, 4, func(d *Deps) {
		d.History = stored
		d.ToolLoading = func() agentapp.ToolLoading { return agentapp.ToolsOnDemand }
	})
	if err != nil {
		t.Fatalf("the turn failed: %v", err)
	}

	second := client.asked[1].Messages
	tail := second[len(second)-4:]
	if tail[0].Search == nil || tail[0].Search.Kind != llm.SearchCall || tail[1].Search == nil || tail[1].Search.Kind != llm.SearchOutput {
		t.Fatalf("the next round replayed %+v, want the search call and its output", tail)
	}
	if tail[2].Call == nil || tail[2].Call.Namespace != "pages" || tail[3].Result == nil || tail[3].Result.CallID != "c1" {
		t.Fatalf("the next round replayed %+v, want the namespaced call and its result", tail[2:])
	}

	var held history
	if err = json.Unmarshal(stored.body, &held); err != nil {
		t.Fatalf("decode the stored history: %v", err)
	}
	called := false
	for _, item := range held.Items {
		if item.Search != nil {
			t.Errorf("the stored history keeps a tool search: %+v", item.Search)
		}
		if item.Call != nil {
			called = true
			if item.Call.Namespace != "" {
				t.Errorf("the stored call keeps the namespace %q the next turn may not load", item.Call.Namespace)
			}
		}
	}
	if !called {
		t.Fatal("the stored history lost the call itself")
	}
}
