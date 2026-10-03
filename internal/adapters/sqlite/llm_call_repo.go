package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/dbx"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	servedTier = `CASE service_tier WHEN '' THEN '` + string(llm.TierDefault) + `' ELSE service_tier END`

	writtenCallColumns = `id, run_id, item_id, step, conversation_id, provider, model, input_tokens, cached_input_tokens,
		cache_write_tokens, output_tokens, reasoning_tokens, usd, latency_ms, status, error_code, service_tier, created_at`
	readCallColumns = `id, run_id, item_id, step, conversation_id, provider, model, input_tokens, cached_input_tokens,
		cache_write_tokens, output_tokens, reasoning_tokens, usd, latency_ms, status, error_code, ` + servedTier + `,
		created_at`
	insertCall = `INSERT INTO llm_calls (` + writtenCallColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	sumColumns = `coalesce(sum(input_tokens), 0), coalesce(sum(cached_input_tokens), 0),
		coalesce(sum(cache_write_tokens), 0), coalesce(sum(output_tokens), 0), coalesce(sum(reasoning_tokens), 0),
		coalesce(sum(usd), 0), count(*)`
	sumByRun          = `SELECT ` + sumColumns + ` FROM llm_calls WHERE run_id = ?`
	sumByConversation = `SELECT ` + sumColumns + ` FROM llm_calls WHERE conversation_id = ?`
	sumAll            = `SELECT ` + sumColumns + ` FROM llm_calls`

	sliceTotals = `sum(CASE WHEN status = ? THEN 1 ELSE 0 END), sum(CASE WHEN status = ? THEN 0 ELSE 1 END),
		sum(input_tokens), sum(cached_input_tokens), sum(cache_write_tokens), sum(output_tokens),
		sum(reasoning_tokens), sum(usd)`
)

type LLMCallRepo struct {
	store *Store
}

func NewLLMCallRepo(store *Store) *LLMCallRepo {
	return &LLMCallRepo{store: store}
}

func (r *LLMCallRepo) Insert(ctx context.Context, call llm.Call) error {
	if err := call.Validate(); err != nil {
		return err
	}
	_, err := execWrite(ctx, r.store.writeFrom(ctx), insertCall, []any{
		call.ID, call.RunID, call.ItemID, call.Step, call.ConversationID, call.Ref.Provider, call.Ref.Model,
		call.Usage.Input, call.Usage.CachedInput, call.Usage.CacheWrite, call.Usage.Output, call.Usage.Reasoning,
		call.USD, call.Latency.Milliseconds(), string(call.Status), call.ErrorCode, string(call.Tier),
		formatTime(call.CreatedAt),
	}, nil, "record the llm call")
	return err
}

func (r *LLMCallRepo) SumByRun(ctx context.Context, runID string) (llm.Spend, error) {
	return r.sum(ctx, sumByRun, runID, "total the llm spend of the run")
}

func (r *LLMCallRepo) SumByConversation(ctx context.Context, conversationID string) (llm.Spend, error) {
	return r.sum(ctx, sumByConversation, conversationID, "total the llm spend of the conversation")
}

func (r *LLMCallRepo) SumAll(ctx context.Context) (llm.Spend, error) {
	return total(r.store.execFrom(ctx).QueryRowContext(ctx, sumAll), "total the llm spend")
}

func (r *LLMCallRepo) sum(ctx context.Context, query, key, message string) (llm.Spend, error) {
	return total(r.store.execFrom(ctx).QueryRowContext(ctx, query, key), message)
}

func total(row *sql.Row, message string) (llm.Spend, error) {
	var spend llm.Spend
	if err := row.Scan(&spend.Usage.Input, &spend.Usage.CachedInput, &spend.Usage.CacheWrite, &spend.Usage.Output,
		&spend.Usage.Reasoning, &spend.USD, &spend.Calls); err != nil {
		return llm.Spend{}, dbx.Convert(err, message)
	}
	spend.Usage.Total = spend.Usage.Input + spend.Usage.Output
	return spend, nil
}

func (r *LLMCallRepo) Aggregate(ctx context.Context, q llm.SpendQuery) ([]llm.SpendSlice, error) {
	purpose, purposeArgs := purposeColumn()
	groups := []string{"purpose", "provider", "model", "tier"}
	step := "'' AS step"
	if q.RunID != "" {
		step = "step"
		groups = append(groups, "step")
	}

	builder := squirrel.Select().
		Column(purpose, purposeArgs...).
		Column("provider").
		Column("model").
		Column(servedTier+" AS tier").
		Column(step).
		Column(sliceTotals, string(llm.CallOK), string(llm.CallOK)).
		From("llm_calls").
		GroupBy(groups...).
		OrderBy(append([]string{"sum(usd) DESC"}, groups...)...)
	if !q.Since.IsZero() {
		builder = builder.Where(squirrel.GtOrEq{"created_at": formatTime(q.Since)})
	}
	if q.RunID != "" {
		builder = builder.Where(squirrel.Eq{"run_id": q.RunID})
	}

	query, args, err := buildQuery(builder, "llm spend")
	if err != nil {
		return nil, err
	}
	return selectAll(ctx, r.store.execFrom(ctx), query, args, scanSpendSlice, "total the llm spend by purpose")
}

func purposeColumn() (expression string, args []any) {
	var column strings.Builder
	args = []any{string(llm.PurposeRun)}
	column.WriteString("CASE WHEN run_id <> '' THEN ?")
	for _, rule := range llm.PurposeRules() {
		column.WriteString(" WHEN step IN (" + placeholders(len(rule.Steps)) + ") THEN ?")
		for _, step := range rule.Steps {
			args = append(args, step)
		}
		args = append(args, string(rule.Purpose))
	}
	column.WriteString(" ELSE ? END AS purpose")
	return column.String(), append(args, string(llm.PurposeOther))
}

func scanSpendSlice(rows *sql.Rows) (llm.SpendSlice, error) {
	var (
		slice         llm.SpendSlice
		purpose, tier string
	)
	if err := rows.Scan(
		&purpose, &slice.Provider, &slice.Model, &tier, &slice.Step, &slice.Calls, &slice.Failed,
		&slice.Input, &slice.CachedInput, &slice.CacheWrite, &slice.Output, &slice.Reasoning, &slice.USD,
	); err != nil {
		return llm.SpendSlice{}, err
	}
	slice.Purpose = llm.Purpose(purpose)
	slice.Tier = llm.ServiceTier(tier)
	return slice, nil
}

func callKeyset(q llm.CallQuery) paging.Keyset[llm.Call] {
	return paging.Keyset[llm.Call]{
		IDColumn: "id",
		ID:       func(c llm.Call) string { return c.ID },
		Keys: []paging.SortKey[llm.Call]{
			paging.TimeKey[llm.Call]("createdAt", "created_at", func(c llm.Call) any { return c.CreatedAt }),
		},
		Desc: q.Desc,
	}
}

func (r *LLMCallRepo) List(ctx context.Context, q llm.CallQuery, page paging.Request) (paging.List[llm.Call], error) {
	builder := squirrel.Select(readCallColumns).From("llm_calls")
	if q.RunID != "" {
		builder = builder.Where(squirrel.Eq{"run_id": q.RunID})
	}
	if q.ConversationID != "" {
		builder = builder.Where(squirrel.Eq{"conversation_id": q.ConversationID})
	}

	keyset := callKeyset(q)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[llm.Call]{}, err
	}
	query, args, err := buildQuery(keyed, "llm calls")
	if err != nil {
		return paging.List[llm.Call]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanCall, "list the llm calls")
	if err != nil {
		return paging.List[llm.Call]{}, err
	}
	return keyset.Cut(rows, page)
}

func scanCall(rows *sql.Rows) (llm.Call, error) {
	var (
		call                    llm.Call
		status, tier, createdAt string
		latency                 int64
	)
	if err := rows.Scan(
		&call.ID, &call.RunID, &call.ItemID, &call.Step, &call.ConversationID, &call.Ref.Provider, &call.Ref.Model,
		&call.Usage.Input, &call.Usage.CachedInput, &call.Usage.CacheWrite, &call.Usage.Output, &call.Usage.Reasoning,
		&call.USD, &latency, &status, &call.ErrorCode, &tier, &createdAt,
	); err != nil {
		return llm.Call{}, err
	}

	call.Usage.Total = call.Usage.Input + call.Usage.Output
	call.Latency = time.Duration(latency) * time.Millisecond
	call.Status = llm.CallStatus(status)
	call.Tier = llm.ServiceTier(tier)

	var err error
	if call.CreatedAt, err = parseTime(createdAt); err != nil {
		return llm.Call{}, err
	}
	return call, nil
}
