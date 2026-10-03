package models_test

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func spentSlices() []llm.SpendSlice {
	return []llm.SpendSlice{
		{
			Purpose: llm.PurposeRun, Provider: "openai", Model: "gpt-5.6-terra", Tier: llm.TierFlex, Calls: 3, Failed: 1,
			Input: 6000, CachedInput: 1000, CacheWrite: 500, Output: 3000, Reasoning: 1500, USD: 0.75,
		},
		{
			Purpose: llm.PurposeChat, Provider: "openai", Model: "gpt-5.6-terra", Tier: llm.TierDefault, Calls: 1,
			Input: 4000, CachedInput: 3000, Output: 1000, USD: 0.25,
		},
	}
}

func TestSpendReport(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		req       models.SpendReportRequest
		wantQuery llm.SpendQuery
		wantDays  int
		wantSince time.Time
		wantRun   string
	}{
		{
			name:      "the last thirty days when no range is named",
			req:       models.SpendReportRequest{},
			wantQuery: llm.SpendQuery{Since: harnessNow.AddDate(0, 0, -30)},
			wantDays:  30, wantSince: harnessNow.AddDate(0, 0, -30),
		},
		{
			name:      "one day",
			req:       models.SpendReportRequest{Days: 1},
			wantQuery: llm.SpendQuery{Since: harnessNow.AddDate(0, 0, -1)},
			wantDays:  1, wantSince: harnessNow.AddDate(0, 0, -1),
		},
		{
			name:      "the longest range",
			req:       models.SpendReportRequest{Days: 366},
			wantQuery: llm.SpendQuery{Since: harnessNow.AddDate(0, 0, -366)},
			wantDays:  366, wantSince: harnessNow.AddDate(0, 0, -366),
		},
		{
			name:      "a run is counted whenever it ran",
			req:       models.SpendReportRequest{RunID: " run-1 "},
			wantQuery: llm.SpendQuery{RunID: "run-1"},
			wantRun:   "run-1",
		},
		{
			name:      "a range beside a run is not applied",
			req:       models.SpendReportRequest{RunID: "run-1", Days: 7},
			wantQuery: llm.SpendQuery{RunID: "run-1"},
			wantRun:   "run-1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			heard := &heardSpend{}
			h := newHarness(t, spend{slices: spentSlices(), heard: heard})
			resp, err := h.service.SpendReport(t.Context(), tc.req)
			if err != nil {
				t.Fatalf("SpendReport: %v", err)
			}
			if heard.aggregate != tc.wantQuery {
				t.Errorf("query = %+v, want %+v", heard.aggregate, tc.wantQuery)
			}
			if resp.Days != tc.wantDays || !resp.Since.Std().Equal(tc.wantSince) || resp.RunID != tc.wantRun {
				t.Errorf("range = %d days since %v for run %q, want %d since %v for %q",
					resp.Days, resp.Since.Std(), resp.RunID, tc.wantDays, tc.wantSince, tc.wantRun)
			}
			if len(resp.Slices) != 2 {
				t.Fatalf("slices = %+v, want both", resp.Slices)
			}
		})
	}
}

func TestSpendReportTotalsTheSlices(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		slices []llm.SpendSlice
		want   models.SpendTotals
	}{
		{
			name:   "a flex run and a chat",
			slices: spentSlices(),
			want: models.SpendTotals{
				Spent: models.Spent{
					Calls: 4, Failed: 1, Input: 10000, CachedInput: 4000, CacheWrite: 500, Output: 4000, Reasoning: 1500,
					USD: 1,
				},
				CachedShare: 0.4, ReasoningShare: 0.375, FlexShare: 0.75,
			},
		},
		{
			name: "calls that cost nothing and read nothing share nothing",
			slices: []llm.SpendSlice{{
				Purpose: llm.PurposeProbe, Provider: "openai", Model: "gpt-5.6-luna", Tier: llm.TierDefault, Failed: 2,
			}},
			want: models.SpendTotals{Spent: models.Spent{Failed: 2}},
		},
		{
			name: "no calls at all",
			want: models.SpendTotals{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t, spend{slices: tc.slices})
			resp, err := h.service.SpendReport(t.Context(), models.SpendReportRequest{})
			if err != nil {
				t.Fatalf("SpendReport: %v", err)
			}
			got := resp.Totals
			if got.Spent != tc.want.Spent {
				t.Errorf("totals = %+v, want %+v", got.Spent, tc.want.Spent)
			}
			for name, pair := range map[string][2]float64{
				"cached":    {got.CachedShare, tc.want.CachedShare},
				"reasoning": {got.ReasoningShare, tc.want.ReasoningShare},
				"flex":      {got.FlexShare, tc.want.FlexShare},
			} {
				if math.Abs(pair[0]-pair[1]) > 1e-9 {
					t.Errorf("%s share = %v, want %v", name, pair[0], pair[1])
				}
			}
			if resp.Slices == nil || len(resp.Slices) != len(tc.slices) {
				t.Errorf("slices = %#v, want one view per slice and never null", resp.Slices)
			}
		})
	}
}

func TestSpendReportSpeaksCamelCase(t *testing.T) {
	t.Parallel()

	h := newHarness(t, spend{slices: spentSlices()[:1]})
	resp, err := h.service.SpendReport(t.Context(), models.SpendReportRequest{RunID: "run-1"})
	if err != nil {
		t.Fatalf("SpendReport: %v", err)
	}
	encoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var shape struct {
		Since  any              `json:"since"`
		Days   float64          `json:"days"`
		RunID  string           `json:"runId"`
		Totals map[string]any   `json:"totals"`
		Slices []map[string]any `json:"slices"`
	}
	if err = json.Unmarshal(encoded, &shape); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{
		"calls", "failed", "input", "cachedInput", "cacheWrite", "output", "reasoning", "usd",
		"cachedShare", "reasoningShare", "flexShare",
	} {
		if _, ok := shape.Totals[key]; !ok {
			t.Errorf("totals carry no %q: %s", key, encoded)
		}
	}
	if len(shape.Slices) != 1 {
		t.Fatalf("slices = %s, want one", encoded)
	}
	want := map[string]any{
		"purpose": "run", "provider": "openai", "model": "gpt-5.6-terra", "tier": "flex", "step": "",
		"calls": 3.0, "failed": 1.0, "input": 6000.0, "cachedInput": 1000.0, "cacheWrite": 500.0, "output": 3000.0,
		"reasoning": 1500.0, "usd": 0.75,
	}
	for key, value := range want {
		if shape.Slices[0][key] != value {
			t.Errorf("slice %q = %v, want %v", key, shape.Slices[0][key], value)
		}
	}
	if shape.Since != nil || shape.Days != 0 || shape.RunID != "run-1" {
		t.Errorf("a run report = %s, want no window and the run", encoded)
	}
}

func TestSpendReportRefusesARangeItCannotRead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		req  models.SpendReportRequest
		code errors.Code
		book spend
	}{
		{name: "a negative range", req: models.SpendReportRequest{Days: -1}, code: errors.Invalid},
		{name: "more than a year", req: models.SpendReportRequest{Days: 367}, code: errors.Invalid},
		{name: "a range beside a run is still checked", req: models.SpendReportRequest{Days: 400, RunID: "run-1"}, code: errors.Invalid},
		{name: "the ledger fails", req: models.SpendReportRequest{}, code: errors.Internal, book: spend{err: errors.New(errors.Internal, "disk")}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t, tc.book)
			_, err := h.service.SpendReport(t.Context(), tc.req)
			if !errors.IsCode(err, tc.code) {
				t.Fatalf("SpendReport error = %v, want %s", err, tc.code)
			}
			if tc.code == errors.Invalid && detail(err, "field") != "days" {
				t.Errorf("field = %v, want days", detail(err, "field"))
			}
		})
	}
}

func TestListCalls(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 10, 3, 9, 30, 0, 0, time.UTC)
	listed := paging.List[llm.Call]{
		Items: []llm.Call{
			{
				ID: "c1", RunID: "run-1", ItemID: "item-1", Step: "generate_body",
				Ref:   llm.ModelRef{Provider: "openai", Model: "gpt-5.6-terra"},
				Usage: llm.Usage{Input: 5000, CachedInput: 1000, CacheWrite: 2000, Output: 900, Reasoning: 400, Total: 5900},
				USD:   0.03, Latency: 1500 * time.Millisecond, Status: llm.CallOK, Tier: llm.TierFlex, CreatedAt: at,
			},
			{
				ID: "c2", ConversationID: "chat-1", Step: llm.StepChat,
				Ref:    llm.ModelRef{Provider: "openai", Model: "gpt-5.6-terra"},
				Status: llm.CallError, ErrorCode: "CANCELLED", Tier: llm.TierDefault, CreatedAt: at,
			},
		},
		Cursors: paging.Cursors{Next: "next"},
		HasMore: true,
	}

	cases := []struct {
		name string
		req  models.ListCallsRequest
		want llm.CallQuery
	}{
		{name: "the newest first when no order is named", req: models.ListCallsRequest{}, want: llm.CallQuery{Desc: true}},
		{
			name: "the oldest first on request",
			req:  models.ListCallsRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "createdAt"}}},
			want: llm.CallQuery{},
		},
		{
			name: "one run's calls",
			req:  models.ListCallsRequest{RunID: " run-1 ", ListRequest: dto.ListRequest{Cursor: "after", Limit: 20}},
			want: llm.CallQuery{RunID: "run-1", Desc: true},
		},
		{
			name: "one conversation's calls",
			req:  models.ListCallsRequest{ConversationID: "chat-1"},
			want: llm.CallQuery{ConversationID: "chat-1", Desc: true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			heard := &heardSpend{}
			h := newHarness(t, spend{listed: listed, heard: heard})
			resp, err := h.service.ListCalls(t.Context(), tc.req)
			if err != nil {
				t.Fatalf("ListCalls: %v", err)
			}
			if heard.calls != tc.want {
				t.Errorf("query = %+v, want %+v", heard.calls, tc.want)
			}
			if heard.page.After != paging.Cursor(tc.req.Cursor) || heard.page.Limit != tc.req.Limit {
				t.Errorf("page = %+v, want the request's cursor and limit", heard.page)
			}
			if resp.Next != "next" || !resp.HasMore || len(resp.Items) != 2 {
				t.Fatalf("list = %+v, want both calls and the next cursor", resp)
			}

			run := resp.Items[0]
			wantRun := models.Call{
				ID: "c1", CreatedAt: dto.NewTime(at), Purpose: string(llm.PurposeRun), Step: "generate_body",
				Provider: "openai", Model: "gpt-5.6-terra", Tier: string(llm.TierFlex),
				Usage: models.Usage{Input: 5000, CachedInput: 1000, CacheWrite: 2000, Output: 900, Reasoning: 400, Total: 5900},
				USD:   0.03, LatencyMs: 1500, Status: string(llm.CallOK), RunID: "run-1", ItemID: "item-1",
			}
			if run != wantRun {
				t.Errorf("run call = %+v, want %+v", run, wantRun)
			}
			chat := resp.Items[1]
			if chat.Purpose != string(llm.PurposeChat) || chat.ConversationID != "chat-1" || chat.ErrorCode != "CANCELLED" ||
				chat.Status != string(llm.CallError) {
				t.Errorf("chat call = %+v, want a failed chat round", chat)
			}
		})
	}
}

func TestListCallsRefusesWhatItCannotList(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		req   models.ListCallsRequest
		code  errors.Code
		field string
		book  spend
	}{
		{
			name: "a run and a conversation at once",
			req:  models.ListCallsRequest{RunID: "run-1", ConversationID: "chat-1"},
			code: errors.Invalid, field: "conversationId",
		},
		{
			name: "an order the ledger does not keep",
			req:  models.ListCallsRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "usd"}}},
			code: errors.Invalid, field: "sort.field",
		},
		{
			name: "the ledger fails",
			code: errors.Internal,
			book: spend{err: errors.New(errors.Internal, "disk")},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t, tc.book)
			_, err := h.service.ListCalls(t.Context(), tc.req)
			if !errors.IsCode(err, tc.code) {
				t.Fatalf("ListCalls error = %v, want %s", err, tc.code)
			}
			if tc.field != "" && detail(err, "field") != tc.field {
				t.Errorf("field = %v, want %s", detail(err, "field"), tc.field)
			}
		})
	}
}
