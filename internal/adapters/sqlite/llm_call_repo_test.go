package sqlite_test

import (
	"slices"
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

	everything, err := repo.SumAll(ctx)
	if err != nil {
		t.Fatalf("SumAll: %v", err)
	}
	if everything.Calls != 6 || everything.Usage.Total != 765 {
		t.Errorf("total spend = %+v, want six calls of 765 tokens", everything)
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

	second, err := repo.List(ctx, llm.CallQuery{RunID: "run-1"}, paging.Request{Limit: 2, After: first.Next})
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

func TestLLMCallRepoKeepsTheUsageDetailAndTheServedTier(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		usage    llm.Usage
		tier     llm.ServiceTier
		wantTier llm.ServiceTier
	}{
		{
			name:  "a flex call with reasoning and a cache write",
			usage: llm.Usage{Input: 4000, CachedInput: 1000, CacheWrite: 2000, Output: 900, Reasoning: 600, Total: 4900},
			tier:  llm.TierFlex, wantTier: llm.TierFlex,
		},
		{
			name:  "a default call",
			usage: llm.Usage{Input: 30, Output: 7, Total: 37},
			tier:  llm.TierDefault, wantTier: llm.TierDefault,
		},
		{
			name:     "a call that names no tier was served at the default one",
			usage:    llm.Usage{Input: 12, Output: 3, Total: 15},
			wantTier: llm.TierDefault,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := sqlite.NewLLMCallRepo(sqlitetest.Open(t))
			call := llmCall("run-1", "", at, 0, 0, 0.125)
			call.Usage = tc.usage
			call.Tier = tc.tier
			if err := repo.Insert(t.Context(), call); err != nil {
				t.Fatalf("Insert: %v", err)
			}

			listed, err := repo.List(t.Context(), llm.CallQuery{RunID: "run-1"}, paging.Request{Limit: 10})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(listed.Items) != 1 {
				t.Fatalf("rows = %d, want 1", len(listed.Items))
			}
			got := listed.Items[0]
			if got.Usage != tc.usage {
				t.Errorf("usage = %+v, want %+v", got.Usage, tc.usage)
			}
			if got.Tier != tc.wantTier {
				t.Errorf("tier = %q, want %q", got.Tier, tc.wantTier)
			}

			spend, err := repo.SumByRun(t.Context(), "run-1")
			if err != nil {
				t.Fatalf("SumByRun: %v", err)
			}
			if spend.Usage != tc.usage {
				t.Errorf("summed usage = %+v, want %+v", spend.Usage, tc.usage)
			}
		})
	}
}

type spendRow struct {
	runID, step, model, conversation string
	tier                             llm.ServiceTier
	status                           llm.CallStatus
	minutesAgo                       int
	usage                            llm.Usage
	usd                              float64
}

func seedSpend(t *testing.T, repo *sqlite.LLMCallRepo, now time.Time, rows []spendRow) {
	t.Helper()

	for i := range rows {
		row := &rows[i]
		call := llm.Call{
			ID: id.New(), RunID: row.runID, Step: row.step, ConversationID: row.conversation,
			Ref:   llm.ModelRef{Provider: "openai", Model: row.model},
			Usage: row.usage, USD: row.usd, Status: row.status, Tier: row.tier,
			CreatedAt: now.Add(-time.Duration(row.minutesAgo) * time.Minute),
		}
		if row.status == llm.CallError {
			call.ErrorCode = "EXTERNAL"
		}
		if err := repo.Insert(t.Context(), call); err != nil {
			t.Fatalf("Insert %+v: %v", row, err)
		}
	}
}

func TestLLMCallRepoAggregate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	const day = 24 * 60
	terra, luna := "gpt-5.6-terra", "gpt-5.6-luna"
	rows := []spendRow{
		{runID: "run-1", step: "generate_body", model: terra, tier: llm.TierFlex, status: llm.CallOK, minutesAgo: 10,
			usage: llm.Usage{Input: 1000, CachedInput: 200, CacheWrite: 300, Output: 2000, Reasoning: 800}, usd: 0.5},
		{runID: "run-1", step: "generate_body", model: terra, tier: llm.TierFlex, status: llm.CallOK, minutesAgo: 9,
			usage: llm.Usage{Input: 1000, Output: 1000, Reasoning: 200}, usd: 0.25},
		{runID: "run-1", step: "generate_body", model: terra, tier: llm.TierFlex, status: llm.CallError, minutesAgo: 8},
		{runID: "run-1", step: "judge", model: luna, tier: llm.TierDefault, status: llm.CallOK, minutesAgo: 7,
			usage: llm.Usage{Input: 500, Output: 100}, usd: 0.125},
		{runID: "run-2", step: "generate_body", model: terra, status: llm.CallOK, minutesAgo: 6,
			usage: llm.Usage{Input: 10, Output: 10}, usd: 0.0625},
		{step: "chat", conversation: "c1", model: terra, status: llm.CallOK, minutesAgo: 5,
			usage: llm.Usage{Input: 20000, CachedInput: 16000, Output: 300}, usd: 0.5},
		{step: "chat", conversation: "c1", model: terra, tier: llm.TierDefault, status: llm.CallError, minutesAgo: 4},
		{step: "title", conversation: "c1", model: luna, status: llm.CallOK, minutesAgo: 3,
			usage: llm.Usage{Input: 40, Output: 8}, usd: 0.03125},
		{step: "test_provider", model: luna, status: llm.CallOK, minutesAgo: 2,
			usage: llm.Usage{Input: 1, Output: 1}, usd: 0},
		{step: "propose_related", model: luna, status: llm.CallOK, minutesAgo: 2,
			usage: llm.Usage{Input: 300, Output: 50}, usd: 0.015625},
		{step: "propose_from_pages", model: luna, status: llm.CallOK, minutesAgo: 1,
			usage: llm.Usage{Input: 100, Output: 50}, usd: 0.015625},
		{step: "judge", model: luna, status: llm.CallOK, minutesAgo: 1,
			usage: llm.Usage{Input: 600, Output: 60}, usd: 0.0078125},
		{model: luna, status: llm.CallOK, minutesAgo: 1, usage: llm.Usage{Input: 5, Output: 5}, usd: 0.00390625},
		{runID: "run-old", step: "generate_body", model: terra, status: llm.CallOK, minutesAgo: 40 * day,
			usage: llm.Usage{Input: 9, Output: 9}, usd: 4},
		{step: "chat", conversation: "c0", model: terra, status: llm.CallOK, minutesAgo: 31 * day,
			usage: llm.Usage{Input: 9, Output: 9}, usd: 2},
	}

	cases := []struct {
		name  string
		query llm.SpendQuery
		want  []llm.SpendSlice
	}{
		{
			name:  "the last thirty days by purpose, model and tier",
			query: llm.SpendQuery{Since: now.Add(-30 * day * time.Minute)},
			want: []llm.SpendSlice{
				{Purpose: llm.PurposeRun, Provider: "openai", Model: terra, Tier: llm.TierFlex, Calls: 2, Failed: 1,
					Input: 2000, CachedInput: 200, CacheWrite: 300, Output: 3000, Reasoning: 1000, USD: 0.75},
				{Purpose: llm.PurposeChat, Provider: "openai", Model: terra, Tier: llm.TierDefault, Calls: 1, Failed: 1,
					Input: 20000, CachedInput: 16000, Output: 300, USD: 0.5},
				{Purpose: llm.PurposeRun, Provider: "openai", Model: luna, Tier: llm.TierDefault, Calls: 1,
					Input: 500, Output: 100, USD: 0.125},
				{Purpose: llm.PurposeRun, Provider: "openai", Model: terra, Tier: llm.TierDefault, Calls: 1,
					Input: 10, Output: 10, USD: 0.0625},
				{Purpose: llm.PurposeGraph, Provider: "openai", Model: luna, Tier: llm.TierDefault, Calls: 2,
					Input: 400, Output: 100, USD: 0.03125},
				{Purpose: llm.PurposeTitle, Provider: "openai", Model: luna, Tier: llm.TierDefault, Calls: 1,
					Input: 40, Output: 8, USD: 0.03125},
				{Purpose: llm.PurposeAudit, Provider: "openai", Model: luna, Tier: llm.TierDefault, Calls: 1,
					Input: 600, Output: 60, USD: 0.0078125},
				{Purpose: llm.PurposeOther, Provider: "openai", Model: luna, Tier: llm.TierDefault, Calls: 1,
					Input: 5, Output: 5, USD: 0.00390625},
				{Purpose: llm.PurposeProbe, Provider: "openai", Model: luna, Tier: llm.TierDefault, Calls: 1,
					Input: 1, Output: 1},
			},
		},
		{
			name:  "one run by step, whenever it ran",
			query: llm.SpendQuery{RunID: "run-1"},
			want: []llm.SpendSlice{
				{Purpose: llm.PurposeRun, Provider: "openai", Model: terra, Tier: llm.TierFlex, Step: "generate_body",
					Calls: 2, Failed: 1, Input: 2000, CachedInput: 200, CacheWrite: 300, Output: 3000, Reasoning: 1000,
					USD: 0.75},
				{Purpose: llm.PurposeRun, Provider: "openai", Model: luna, Tier: llm.TierDefault, Step: "judge",
					Calls: 1, Input: 500, Output: 100, USD: 0.125},
			},
		},
		{
			name:  "an old run outside any window",
			query: llm.SpendQuery{RunID: "run-old"},
			want: []llm.SpendSlice{
				{Purpose: llm.PurposeRun, Provider: "openai", Model: terra, Tier: llm.TierDefault, Step: "generate_body",
					Calls: 1, Input: 9, Output: 9, USD: 4},
			},
		},
		{
			name:  "a run and a window together",
			query: llm.SpendQuery{RunID: "run-old", Since: now.Add(-day * time.Minute)},
			want:  []llm.SpendSlice{},
		},
		{
			name:  "a run nobody started",
			query: llm.SpendQuery{RunID: "run-missing"},
			want:  []llm.SpendSlice{},
		},
	}

	repo := sqlite.NewLLMCallRepo(sqlitetest.Open(t))
	seedSpend(t, repo, now, rows)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := repo.Aggregate(t.Context(), tc.query)
			if err != nil {
				t.Fatalf("Aggregate: %v", err)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("Aggregate(%+v)\n got  %+v\n want %+v", tc.query, got, tc.want)
			}
		})
	}
}

func TestLLMCallRepoAggregateDerivesThePurposeAsTheDomainDoes(t *testing.T) {
	t.Parallel()

	steps := []string{"", "generate_body", "unknown_step"}
	for _, rule := range llm.PurposeRules() {
		steps = append(steps, rule.Steps...)
	}

	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	for _, runID := range []string{"", "run-1"} {
		for _, step := range steps {
			t.Run(runID+"/"+step, func(t *testing.T) {
				t.Parallel()

				repo := sqlite.NewLLMCallRepo(sqlitetest.Open(t))
				seedSpend(t, repo, now, []spendRow{{runID: runID, step: step, model: "gpt-5.6-luna", status: llm.CallOK}})

				got, err := repo.Aggregate(t.Context(), llm.SpendQuery{})
				if err != nil {
					t.Fatalf("Aggregate: %v", err)
				}
				if len(got) != 1 {
					t.Fatalf("slices = %+v, want one", got)
				}
				if want := llm.PurposeOf(runID, step); got[0].Purpose != want {
					t.Errorf("purpose of run %q step %q = %q, want %q", runID, step, got[0].Purpose, want)
				}
			})
		}
	}
}
