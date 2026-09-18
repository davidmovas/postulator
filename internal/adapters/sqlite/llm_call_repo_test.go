package sqlite_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func llmCall(runID, conversationID string, at time.Time, input, output int, usd float64) llm.Call {
	return llm.Call{
		ID:             id.New(),
		RunID:          runID,
		ItemID:         "item",
		Step:           "generate_body",
		ConversationID: conversationID,
		Ref:            llm.ModelRef{Provider: "openai", Model: "gpt-5.6-terra"},
		Usage:          llm.Usage{Input: input, Output: output, Total: input + output},
		USD:            usd,
		Latency:        250 * time.Millisecond,
		Status:         llm.CallOK,
		CreatedAt:      at,
	}
}

func TestLLMCallRepo(t *testing.T) {
	t.Parallel()

	repo := sqlite.NewLLMCallRepo(sqlitetest.Open(t))
	ctx := t.Context()
	at := time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC)

	for i := range 5 {
		call := llmCall("run-1", "chat-1", at.Add(time.Duration(i)*time.Minute), 100, 50, 0.5)
		if err := repo.Insert(ctx, call); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}
	other := llmCall("run-2", "chat-2", at, 10, 5, 0.25)
	other.Status = llm.CallError
	other.ErrorCode = string(llm.CallError)
	if err := repo.Insert(ctx, other); err != nil {
		t.Fatalf("Insert other: %v", err)
	}

	spend, err := repo.SumByRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("SumByRun: %v", err)
	}
	if spend.Calls != 5 || spend.Usage.Input != 500 || spend.Usage.Output != 250 || spend.Usage.Total != 750 {
		t.Errorf("run spend = %+v, want five calls of 100/50", spend)
	}
	if spend.USD < 2.49 || spend.USD > 2.51 {
		t.Errorf("run spend usd = %v, want 2.5", spend.USD)
	}

	conversation, err := repo.SumByConversation(ctx, "chat-2")
	if err != nil {
		t.Fatalf("SumByConversation: %v", err)
	}
	if conversation.Calls != 1 || conversation.Usage.Total != 15 {
		t.Errorf("conversation spend = %+v, want one call of 10/5", conversation)
	}

	empty, err := repo.SumByRun(ctx, "run-missing")
	if err != nil {
		t.Fatalf("SumByRun missing: %v", err)
	}
	if empty.Calls != 0 || empty.USD != 0 {
		t.Errorf("missing run spend = %+v, want zero", empty)
	}

	first, err := repo.List(ctx, llm.CallQuery{RunID: "run-1"}, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(first.Items) != 2 || !first.HasMore {
		t.Fatalf("first page = %d items, hasMore %t, want 2 and true", len(first.Items), first.HasMore)
	}
	if first.Items[0].Latency != 250*time.Millisecond || first.Items[0].Ref.Provider != "openai" {
		t.Errorf("first item = %+v, want the stored latency and reference", first.Items[0])
	}

	second, err := repo.List(ctx, llm.CallQuery{RunID: "run-1"}, paging.Request{Limit: 2, After: first.Cursors.Next})
	if err != nil {
		t.Fatalf("List page two: %v", err)
	}
	if len(second.Items) != 2 || second.Items[0].ID == first.Items[0].ID {
		t.Errorf("second page = %+v, want the next two calls", second.Items)
	}

	descending, err := repo.List(ctx, llm.CallQuery{ConversationID: "chat-1", Desc: true}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List descending: %v", err)
	}
	if len(descending.Items) != 5 || !descending.Items[0].CreatedAt.After(descending.Items[4].CreatedAt) {
		t.Errorf("descending page = %+v, want newest first", descending.Items)
	}
}
