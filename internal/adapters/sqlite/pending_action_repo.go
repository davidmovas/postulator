package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	pendingActionColumns = `id, conversation_id, tool, args, summary, status, result, error, created_at, updated_at`
	insertPendingAction  = `INSERT INTO pending_actions (` + pendingActionColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	selectPendingAction     = `SELECT ` + pendingActionColumns + ` FROM pending_actions WHERE id = ?`
	transitionPendingAction = `UPDATE pending_actions SET status = ?, result = ?, error = ?, updated_at = ?
		WHERE id = ? AND status = ?`
)

type PendingActionRepo struct {
	store *Store
}

func NewPendingActionRepo(store *Store) *PendingActionRepo {
	return &PendingActionRepo{store: store}
}

func pendingActionNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "pending action not found").WithDetail("actionId", id)
}

func (r *PendingActionRepo) Insert(ctx context.Context, a agent.PendingAction) error {
	_, err := execWrite(ctx, r.store.writeFrom(ctx), insertPendingAction, []any{
		a.ID, a.ConversationID, a.Tool, string(payloadOf(a.Args)), a.Summary, string(a.Status),
		string(a.Result), a.Error, formatTime(a.CreatedAt), formatTime(a.UpdatedAt),
	}, errors.New(errors.Conflict, "a pending action with this id already exists"), "insert the pending action")
	return err
}

func (r *PendingActionRepo) Get(ctx context.Context, id string) (agent.PendingAction, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectPendingAction, []any{id}, scanPendingAction,
		pendingActionNotFound(id), "read the pending action")
}

func (r *PendingActionRepo) Transition(ctx context.Context, id string, from, to agent.ActionStatus,
	result json.RawMessage, failure string, now time.Time) (bool, error) {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), transitionPendingAction,
		[]any{string(to), string(result), failure, formatTime(now), id, string(from)}, nil,
		"settle the pending action")
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *PendingActionRepo) List(ctx context.Context, q agent.ActionQuery, page paging.Request) (paging.List[agent.PendingAction], error) {
	builder := squirrel.Select(pendingActionColumns).From("pending_actions")
	if q.ConversationID != "" {
		builder = builder.Where(squirrel.Eq{"conversation_id": q.ConversationID})
	}
	if q.Status != nil {
		builder = builder.Where(squirrel.Eq{"status": string(*q.Status)})
	}

	keyset := pendingActionKeyset(q.Desc)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[agent.PendingAction]{}, err
	}
	query, args, err := buildQuery(keyed, "pending actions")
	if err != nil {
		return paging.List[agent.PendingAction]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanPendingAction, "list the pending actions")
	if err != nil {
		return paging.List[agent.PendingAction]{}, err
	}
	return keyset.Cut(rows, page)
}

func pendingActionKeyset(desc bool) paging.Keyset[agent.PendingAction] {
	return paging.Keyset[agent.PendingAction]{
		IDColumn: "id",
		ID:       func(a agent.PendingAction) string { return a.ID },
		Keys: []paging.SortKey[agent.PendingAction]{
			paging.TimeKey[agent.PendingAction]("createdAt", "created_at", func(a agent.PendingAction) any { return a.CreatedAt }),
		},
		Desc: desc,
	}
}

func scanPendingAction(rows *sql.Rows) (agent.PendingAction, error) {
	var (
		a                    agent.PendingAction
		args, result, status string
		createdAt, updatedAt string
	)
	if err := rows.Scan(&a.ID, &a.ConversationID, &a.Tool, &args, &a.Summary, &status, &result, &a.Error,
		&createdAt, &updatedAt); err != nil {
		return agent.PendingAction{}, err
	}

	a.Args = json.RawMessage(args)
	a.Status = agent.ActionStatus(status)
	if result != "" {
		a.Result = json.RawMessage(result)
	}

	var err error
	if a.CreatedAt, err = parseTime(createdAt); err != nil {
		return agent.PendingAction{}, err
	}
	if a.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return agent.PendingAction{}, err
	}
	return a, nil
}
