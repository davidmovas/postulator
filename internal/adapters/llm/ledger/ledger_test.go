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
	published []events.Type
	mu        sync.Mutex
}

func (r *recorder) Publish(eventType events.Type, _ any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.published = append(r.published, eventType)
	return nil
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.published)
}

type catalog struct{}

func (catalog) Lookup(_ context.Context, ref llm.ModelRef) (llm.ModelInfo, error) {
	if ref.Model == "ghost" {
		return llm.ModelInfo{}, errors.New(errors.NotFound, "no such model")
	}
	return llm.ModelInfo{Ref: ref, InputUSDPerM: 1000000, OutputUSDPerM: 2000000}, nil
}

func newLedger(t *testing.T) (*ledger.Ledger, *sqlite.LLMCallRepo, *recorder) {
	t.Helper()

	repo := sqlite.NewLLMCallRepo(sqlitetest.Open(t))
	events := &recorder{}
	book := ledger.New(fake.New(), repo, catalog{}, events, clock.NewFake(time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC)))
	return book, repo, events
}

func request(model, prompt string) port.Request {
	return port.Request{
		Ref:      llm.ModelRef{Provider: "openai", Model: model},
		Messages: []port.Message{{Role: port.RoleUser, Text: prompt}},
		Meta:     port.CallMeta{RunID: "run-1", ItemID: "item-1", Step: "generate_body", ConversationID: "chat-1"},
	}
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

	calls, err := repo.List(t.Context(), llm.CallQuery{RunID: "run-1"}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(calls.Items) != 1 {
		t.Fatalf("rows = %d, want 1", len(calls.Items))
	}

	row := calls.Items[0]
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
	if published.count() != 1 {
		t.Errorf("events = %d, want one llm.usage", published.count())
	}
}

func TestLedgerRecordsAFailure(t *testing.T) {
	t.Parallel()

	book, repo, published := newLedger(t)
	if _, err := book.Complete(t.Context(), request("gpt-5.6-luna", "ERROR: EXTERNAL")); !errors.IsCode(err, errors.External) {
		t.Fatalf("Complete error = %v, want %s", err, errors.External)
	}

	calls, err := repo.List(t.Context(), llm.CallQuery{RunID: "run-1"}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(calls.Items) != 1 {
		t.Fatalf("rows = %d, want 1", len(calls.Items))
	}
	if calls.Items[0].Status != llm.CallError || calls.Items[0].ErrorCode != string(errors.External) {
		t.Errorf("row = %+v, want an error row carrying the code", calls.Items[0])
	}
	if published.count() != 0 {
		t.Errorf("events = %d, want none for a failed call", published.count())
	}
}

func TestLedgerPricesAnUnknownModelAtZero(t *testing.T) {
	t.Parallel()

	book, repo, _ := newLedger(t)
	if _, err := book.Complete(t.Context(), request("ghost", "ANSWER: hello")); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	calls, err := repo.List(t.Context(), llm.CallQuery{RunID: "run-1"}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if calls.Items[0].USD != 0 {
		t.Errorf("usd = %v, want zero for a model with no price", calls.Items[0].USD)
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
}
