package ledger_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/llm/ledger"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/events"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type recorder struct {
	published []events.LLMUsagePayload
	mu        sync.Mutex
}

func (r *recorder) Publish(eventType events.Type, payload any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if usage, ok := payload.(events.LLMUsagePayload); ok && eventType == events.LLMUsage {
		r.published = append(r.published, usage)
	}
	return nil
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.published)
}

func (r *recorder) last() events.LLMUsagePayload {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.published) == 0 {
		return events.LLMUsagePayload{}
	}
	return r.published[len(r.published)-1]
}

type catalog struct{}

var flexTerra = llm.ModelInfo{
	Ref:          llm.ModelRef{Provider: "openai", Model: "gpt-5.6-terra"},
	InputUSDPerM: 2, CachedInputUSDPerM: 0.2, CacheWriteUSDPerM: 2.5, OutputUSDPerM: 12,
	FlexInputUSDPerM: 1, FlexCachedInputUSDPerM: 0.1, FlexCacheWriteUSDPerM: 1.25, FlexOutputUSDPerM: 6,
}

func (catalog) Lookup(_ context.Context, ref llm.ModelRef) (llm.ModelInfo, error) {
	switch ref.Model {
	case "ghost":
		return llm.ModelInfo{}, errors.New(errors.NotFound, "no such model")
	case flexTerra.Ref.Model:
		return flexTerra, nil
	default:
		return llm.ModelInfo{Ref: ref, InputUSDPerM: 1000000, OutputUSDPerM: 2000000}, nil
	}
}

type announcing struct {
	*sqlite.LLMCallRepo
	inserted chan llm.Call
}

func (s announcing) Insert(ctx context.Context, call llm.Call) error {
	err := s.LLMCallRepo.Insert(ctx, call)
	s.inserted <- call
	return err
}

func newLedger(t *testing.T) (*ledger.Ledger, *sqlite.LLMCallRepo, *recorder) {
	t.Helper()
	return newLedgerOver(t, fake.New())
}

func newLedgerOver(t *testing.T, next port.Client) (*ledger.Ledger, *sqlite.LLMCallRepo, *recorder) {
	t.Helper()

	repo := sqlite.NewLLMCallRepo(sqlitetest.Open(t))
	published := &recorder{}
	book := ledger.New(next, repo, catalog{}, published, clock.NewFake(time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC)))
	return book, repo, published
}

func request(model, prompt string) port.Request {
	return port.Request{
		Ref:      llm.ModelRef{Provider: "openai", Model: model},
		Messages: []port.Message{{Role: port.RoleUser, Text: prompt}},
		Meta:     port.CallMeta{RunID: "run-1", ItemID: "item-1", Step: "generate_body", ConversationID: "chat-1"},
	}
}

func onlyRow(t *testing.T, repo *sqlite.LLMCallRepo) llm.Call {
	t.Helper()

	calls, err := repo.List(t.Context(), llm.CallQuery{RunID: "run-1"}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(calls.Items) != 1 {
		t.Fatalf("rows = %d, want 1", len(calls.Items))
	}
	return calls.Items[0]
}

type served struct {
	response port.Response
	deltas   func(ctx context.Context) <-chan port.Delta
}

func (s served) Complete(ctx context.Context, _ port.Request) (port.Response, error) {
	if err := ctx.Err(); err != nil {
		return port.Response{}, errors.New(errors.Cancelled, "the call was cancelled").WithInternal(err)
	}
	return s.response, nil
}

func (s served) Stream(ctx context.Context, _ port.Request) (<-chan port.Delta, error) {
	return s.deltas(ctx), nil
}

func TestLedgerRecordsACompletion(t *testing.T) {
	t.Parallel()

	book, repo, published := newLedger(t)
	resp, err := book.Complete(t.Context(), request("gpt-5.6-luna", "ANSWER: Koffein"))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Text != "Koffein" {
		t.Fatalf("text = %q, want the scripted answer", resp.Text)
	}

	row := onlyRow(t, repo)
	if row.Status != llm.CallOK || row.ErrorCode != "" {
		t.Errorf("row = %+v, want an ok row", row)
	}
	if row.Step != "generate_body" || row.ConversationID != "chat-1" || row.ItemID != "item-1" {
		t.Errorf("row = %+v, want the call metadata", row)
	}
	if row.Usage != resp.Usage {
		t.Errorf("row usage = %+v, want %+v", row.Usage, resp.Usage)
	}
	if row.USD <= 0 {
		t.Errorf("row usd = %v, want a positive cost", row.USD)
	}
	if row.Tier != llm.TierDefault {
		t.Errorf("row tier = %q, want the default tier for a client that reports none", row.Tier)
	}
	if published.count() != 1 {
		t.Errorf("events = %d, want one llm.usage", published.count())
	}
}

func TestLedgerRecordsTheServedTierAndTheUsageDetail(t *testing.T) {
	t.Parallel()

	usage := llm.Usage{Input: 10000, CachedInput: 4000, CacheWrite: 3000, Output: 2000, Reasoning: 1500, Total: 12000}
	cases := []struct {
		name     string
		tier     llm.ServiceTier
		wantTier llm.ServiceTier
	}{
		{name: "a flex answer is booked at the flex prices", tier: llm.TierFlex, wantTier: llm.TierFlex},
		{name: "a default answer is booked at the standard prices", tier: llm.TierDefault, wantTier: llm.TierDefault},
		{name: "an answer that names no tier was served at the default one", wantTier: llm.TierDefault},
		{
			name: "a tier the domain does not know is booked as the default one",
			tier: llm.ServiceTier("priority"), wantTier: llm.TierDefault,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			book, repo, published := newLedgerOver(t, served{response: port.Response{Text: "ok", Usage: usage, Tier: tc.tier}})
			if _, err := book.Complete(t.Context(), request(flexTerra.Ref.Model, "hello")); err != nil {
				t.Fatalf("Complete: %v", err)
			}

			row := onlyRow(t, repo)
			if row.Tier != tc.wantTier {
				t.Errorf("tier = %q, want %q", row.Tier, tc.wantTier)
			}
			if row.Usage != usage {
				t.Errorf("usage = %+v, want %+v", row.Usage, usage)
			}
			if want := llm.Cost(usage, flexTerra, tc.wantTier); row.USD != want {
				t.Errorf("usd = %v, want %v", row.USD, want)
			}

			heard := published.last()
			if heard.Tier != string(tc.wantTier) || heard.ReasoningTokens != usage.Reasoning ||
				heard.CacheWriteTokens != usage.CacheWrite || heard.PromptTokens != usage.Input ||
				heard.CompletionTokens != usage.Output || heard.USD != row.USD {
				t.Errorf("llm.usage = %+v, want the row's tier, tokens and cost", heard)
			}
		})
	}
}

func TestLedgerRecordsAFailure(t *testing.T) {
	t.Parallel()

	book, repo, published := newLedger(t)
	if _, err := book.Complete(t.Context(), request("gpt-5.6-luna", "ERROR: EXTERNAL")); !errors.IsCode(err, errors.External) {
		t.Fatalf("Complete error = %v, want %s", err, errors.External)
	}

	row := onlyRow(t, repo)
	if row.Status != llm.CallError || row.ErrorCode != string(errors.External) {
		t.Errorf("row = %+v, want an error row carrying the code", row)
	}
	if published.count() != 0 {
		t.Errorf("events = %d, want none for a failed call", published.count())
	}
}

func TestLedgerRecordsACallItsCallerCancelled(t *testing.T) {
	t.Parallel()

	book, repo, _ := newLedgerOver(t, served{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := book.Complete(ctx, request(flexTerra.Ref.Model, "hello")); !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("Complete error = %v, want %s", err, errors.Cancelled)
	}

	row := onlyRow(t, repo)
	if row.Status != llm.CallError || row.ErrorCode != string(errors.Cancelled) {
		t.Errorf("row = %+v, want a cancelled row", row)
	}
}

func TestLedgerPricesAnUnknownModelAtZero(t *testing.T) {
	t.Parallel()

	book, repo, _ := newLedger(t)
	if _, err := book.Complete(t.Context(), request("ghost", "ANSWER: hello")); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if row := onlyRow(t, repo); row.USD != 0 {
		t.Errorf("usd = %v, want zero for a model with no price", row.USD)
	}
}

func TestLedgerRecordsAStream(t *testing.T) {
	t.Parallel()

	book, _, published := newLedger(t)
	deltas, err := book.Stream(t.Context(), request("gpt-5.6-luna", "ANSWER: Koffein und Powder"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	for delta := range deltas {
		if delta.Err != nil {
			t.Fatalf("delta error: %v", delta.Err)
		}
	}

	spend, err := book.SumByConversation(t.Context(), "chat-1")
	if err != nil {
		t.Fatalf("SumByConversation: %v", err)
	}
	if spend.Calls != 1 || spend.Usage.Output == 0 {
		t.Errorf("spend = %+v, want one recorded stream", spend)
	}
	if published.count() != 1 {
		t.Errorf("events = %d, want one llm.usage", published.count())
	}
}

func TestLedgerRecordsTheTierAStreamWasServedAt(t *testing.T) {
	t.Parallel()

	usage := llm.Usage{Input: 2000, CachedInput: 1000, Output: 400, Reasoning: 100, Total: 2400}
	book, repo, published := newLedgerOver(t, served{deltas: func(context.Context) <-chan port.Delta {
		out := make(chan port.Delta, 3)
		out <- port.Delta{Text: "Kof"}
		out <- port.Delta{Text: "fein"}
		out <- port.Delta{Done: true, Usage: &usage, Finish: port.FinishStop, Tier: llm.TierFlex}
		close(out)
		return out
	}})

	deltas, err := book.Stream(t.Context(), request(flexTerra.Ref.Model, "hello"))
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	for delta := range deltas {
		if delta.Err != nil {
			t.Fatalf("delta error: %v", delta.Err)
		}
	}

	row := onlyRow(t, repo)
	if row.Tier != llm.TierFlex || row.Usage != usage || row.Status != llm.CallOK {
		t.Errorf("row = %+v, want an ok flex row with the final usage", row)
	}
	if want := llm.Cost(usage, flexTerra, llm.TierFlex); row.USD != want {
		t.Errorf("usd = %v, want %v", row.USD, want)
	}
	if heard := published.last(); heard.Tier != string(llm.TierFlex) || heard.ReasoningTokens != 100 {
		t.Errorf("llm.usage = %+v, want the flex tier and the reasoning tokens", heard)
	}
}

func TestLedgerRecordsAStreamItsCallerCancelled(t *testing.T) {
	t.Parallel()

	arrived := llm.Usage{Input: 3000, Output: 40, Total: 3040}
	cases := []struct {
		name string
		sent []port.Delta
	}{
		{
			name: "the caller stops reading while a part waits to be passed on",
			sent: []port.Delta{{Text: "Kof"}, {Usage: &arrived, Tier: llm.TierFlex}},
		},
		{
			name: "the client closes the stream without a last part when its caller cancels",
			sent: []port.Delta{{Text: "Kof", Usage: &arrived, Tier: llm.TierFlex}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := sqlite.NewLLMCallRepo(sqlitetest.Open(t))
			store := announcing{LLMCallRepo: repo, inserted: make(chan llm.Call, 1)}
			upstream := served{deltas: func(ctx context.Context) <-chan port.Delta {
				out := make(chan port.Delta)
				go func() {
					defer close(out)
					for _, delta := range tc.sent {
						out <- delta
					}
					<-ctx.Done()
				}()
				return out
			}}
			book := ledger.New(upstream, store, catalog{}, &recorder{},
				clock.NewFake(time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC)))

			ctx, cancel := context.WithCancel(t.Context())
			deltas, err := book.Stream(ctx, request(flexTerra.Ref.Model, "hello"))
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}
			if first := <-deltas; first.Text != "Kof" {
				t.Fatalf("first delta = %+v, want the first words", first)
			}
			cancel()

			select {
			case <-store.inserted:
			case <-time.After(10 * time.Second):
				t.Fatal("a cancelled stream wrote no row to the ledger")
			}
			passed := 0
			for range deltas {
				passed++
			}
			if passed != 0 {
				t.Errorf("%d more parts reached a caller that cancelled, want none", passed)
			}

			row := onlyRow(t, repo)
			if row.Status != llm.CallError || row.ErrorCode != string(errors.Cancelled) {
				t.Errorf("row = %+v, want a cancelled row", row)
			}
			if row.Usage != arrived || row.Tier != llm.TierFlex {
				t.Errorf("row = %+v, want the usage and the tier that arrived before the cancel", row)
			}
			if want := llm.Cost(arrived, flexTerra, llm.TierFlex); row.USD != want {
				t.Errorf("usd = %v, want %v", row.USD, want)
			}
		})
	}
}

func TestLedgerRecordsAStreamThatNeverStarted(t *testing.T) {
	t.Parallel()

	book, _, _ := newLedger(t)
	if _, err := book.Stream(t.Context(), request("gpt-5.6-luna", "ERROR: RATE_LIMITED")); !errors.IsCode(err, errors.RateLimited) {
		t.Fatalf("Stream error = %v, want %s", err, errors.RateLimited)
	}

	spend, err := book.SumByRun(t.Context(), "run-1")
	if err != nil {
		t.Fatalf("SumByRun: %v", err)
	}
	if spend.Calls != 1 {
		t.Errorf("spend = %+v, want the failed attempt recorded", spend)
	}
}

func TestLedgerSumsAcrossAttempts(t *testing.T) {
	t.Parallel()

	book, _, _ := newLedger(t)
	for range 3 {
		if _, err := book.Complete(t.Context(), request("gpt-5.6-luna", "ANSWER: Koffein")); err != nil {
			t.Fatalf("Complete: %v", err)
		}
	}

	spend, err := book.SumByRun(t.Context(), "run-1")
	if err != nil {
		t.Fatalf("SumByRun: %v", err)
	}
	if spend.Calls != 3 {
		t.Errorf("spend = %+v, want three attempts", spend)
	}

	listed, err := book.List(t.Context(), llm.CallQuery{ConversationID: "chat-1"}, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed.Items) != 2 || !listed.HasMore {
		t.Errorf("page = %d items, hasMore %t, want 2 and true", len(listed.Items), listed.HasMore)
	}

	grouped, err := book.Aggregate(t.Context(), llm.SpendQuery{RunID: "run-1"})
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	if len(grouped) != 1 || grouped[0].Calls != 3 || grouped[0].Step != "generate_body" || grouped[0].Purpose != llm.PurposeRun {
		t.Errorf("slices = %+v, want the three attempts under the run's step", grouped)
	}
}
