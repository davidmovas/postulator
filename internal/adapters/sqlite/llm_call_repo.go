package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/dbx"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	callColumns = `id, run_id, item_id, step, conversation_id, provider, model, input_tokens, output_tokens, usd,
		latency_ms, status, error_code, created_at`
	insertCall = `INSERT INTO llm_calls (` + callColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	sumByRun   = `SELECT coalesce(sum(input_tokens), 0), coalesce(sum(output_tokens), 0), coalesce(sum(usd), 0), count(*)
		FROM llm_calls WHERE run_id = ?`
	sumByConversation = `SELECT coalesce(sum(input_tokens), 0), coalesce(sum(output_tokens), 0), coalesce(sum(usd), 0), count(*)
		FROM llm_calls WHERE conversation_id = ?`
)

type LLMCallRepo struct {
	store *Store
}

func NewLLMCallRepo(store *Store) *LLMCallRepo {
	return &LLMCallRepo{store: store}
}

func (r *LLMCallRepo) Insert(ctx context.Context, call llm.Call) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), insertCall, []any{
		call.ID, call.RunID, call.ItemID, call.Step, call.ConversationID, call.Ref.Provider, call.Ref.Model,
		call.Usage.Input, call.Usage.Output, call.USD, call.Latency.Milliseconds(), string(call.Status),
		call.ErrorCode, formatTime(call.CreatedAt),
	}, nil, "record the llm call")
	return err
}

func (r *LLMCallRepo) SumByRun(ctx context.Context, runID string) (llm.Spend, error) {
	return r.sum(ctx, sumByRun, runID, "total the llm spend of the run")
}

func (r *LLMCallRepo) SumByConversation(ctx context.Context, conversationID string) (llm.Spend, error) {
	return r.sum(ctx, sumByConversation, conversationID, "total the llm spend of the conversation")
}

func (r *LLMCallRepo) sum(ctx context.Context, query, key, message string) (llm.Spend, error) {
	var spend llm.Spend
	row := r.store.execFrom(ctx).QueryRowContext(ctx, query, key)
	if err := row.Scan(&spend.Usage.Input, &spend.Usage.Output, &spend.USD, &spend.Calls); err != nil {
		return llm.Spend{}, dbx.Convert(err, message)
	}
	spend.Usage.Total = spend.Usage.Input + spend.Usage.Output
	return spend, nil
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
	builder := squirrel.Select(callColumns).From("llm_calls")
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
		call      llm.Call
		status    string
		latency   int64
		createdAt string
	)
	if err := rows.Scan(
		&call.ID, &call.RunID, &call.ItemID, &call.Step, &call.ConversationID, &call.Ref.Provider, &call.Ref.Model,
		&call.Usage.Input, &call.Usage.Output, &call.USD, &latency, &status, &call.ErrorCode, &createdAt,
	); err != nil {
		return llm.Call{}, err
	}

	call.Usage.Total = call.Usage.Input + call.Usage.Output
	call.Latency = time.Duration(latency) * time.Millisecond
	call.Status = llm.CallStatus(status)

	var err error
	if call.CreatedAt, err = parseTime(createdAt); err != nil {
		return llm.Call{}, err
	}
	return call, nil
}
