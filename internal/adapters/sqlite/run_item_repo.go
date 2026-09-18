package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	itemColumns = `id, run_id, site_id, target_id, status, current_step, attempts, advance_seq, checkpoint,
		lease_until, wake_at, pause_reason, error, created_at, updated_at, finished_at`
	insertItem = `INSERT INTO run_items (` + itemColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	selectItem = `SELECT ` + itemColumns + ` FROM run_items WHERE id = ?`
	claimItem  = `UPDATE run_items SET status = 'running', advance_seq = advance_seq + 1, lease_until = ?,
		wake_at = NULL, updated_at = ? WHERE id = ? AND advance_seq = ?
		AND status IN ('pending', 'running', 'waiting')`
	persistItem = `UPDATE run_items SET status = ?, current_step = ?, attempts = ?, checkpoint = ?, lease_until = ?,
		wake_at = ?, pause_reason = ?, error = ?, updated_at = ?, finished_at = ? WHERE id = ? AND advance_seq = ?`
	selectItemsByRun    = `SELECT ` + itemColumns + ` FROM run_items WHERE run_id = ? ORDER BY created_at, id`
	selectItemsByTarget = `SELECT ` + itemColumns + ` FROM run_items WHERE target_id = ?
		ORDER BY created_at DESC, id DESC LIMIT ?`
	selectDueItems = `SELECT ` + itemColumns + ` FROM run_items
		WHERE status = 'waiting' AND wake_at IS NOT NULL AND wake_at <= ? ORDER BY wake_at, id LIMIT ?`
	selectStalledItems = `SELECT ` + itemColumns + ` FROM run_items
		WHERE status = 'running' AND lease_until IS NOT NULL AND lease_until <= ? ORDER BY lease_until, id LIMIT ?`
	selectRunnableItems = `SELECT i.id, i.run_id, i.site_id, i.target_id, i.status, i.current_step, i.attempts,
		i.advance_seq, i.checkpoint, i.lease_until, i.wake_at, i.pause_reason, i.error, i.created_at, i.updated_at,
		i.finished_at
		FROM run_items i JOIN runs r ON r.id = i.run_id
		WHERE r.status IN ('pending', 'running') AND i.status = 'pending'
		AND (i.lease_until IS NULL OR i.lease_until <= ?) ORDER BY i.created_at, i.id LIMIT ?`
	requeueItem = `UPDATE run_items SET status = 'pending', advance_seq = advance_seq + 1, lease_until = NULL,
		wake_at = NULL, updated_at = ? WHERE id = ? AND advance_seq = ? AND status = ?`
	countItemsByStatus = `SELECT status, count(*) FROM run_items WHERE run_id = ? GROUP BY status`
	stopItemsOfRun     = `UPDATE run_items SET status = ?, pause_reason = ?, advance_seq = advance_seq + 1,
		lease_until = NULL, wake_at = NULL, updated_at = ?, finished_at = ? WHERE run_id = ? AND status IN `
	resumeItemsOfRun = `UPDATE run_items SET status = 'pending', pause_reason = '', advance_seq = advance_seq + 1,
		lease_until = NULL, wake_at = NULL, updated_at = ? WHERE run_id = ? AND status = 'paused'`
)

var errNotClaimed = errors.New(errors.Conflict, "the run item moved on before it could be claimed")

type RunItemRepo struct {
	store *Store
}

func NewRunItemRepo(store *Store) *RunItemRepo {
	return &RunItemRepo{store: store}
}

func itemNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "run item not found").WithDetail("itemId", id)
}

func (r *RunItemRepo) Insert(ctx context.Context, item run.Item) error {
	checkpoint, err := item.Checkpoint.Encode()
	if err != nil {
		return err
	}

	_, err = execWrite(ctx, r.store.writeFrom(ctx), insertItem, []any{
		item.ID, item.RunID, item.SiteID, item.TargetID, string(item.Status), item.CurrentStep, item.Attempts,
		item.AdvanceSeq, checkpoint, nullTime(item.LeaseUntil), nullTime(item.WakeAt), string(item.PauseReason), item.Error,
		formatTime(item.CreatedAt), formatTime(item.UpdatedAt), nullTime(item.FinishedAt),
	}, errors.New(errors.Conflict, "a run item with this id already exists"), "insert the run item")
	return err
}

func (r *RunItemRepo) Get(ctx context.Context, id string) (run.Item, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectItem, []any{id}, scanItem, itemNotFound(id), "read the run item")
}

func (r *RunItemRepo) Claim(ctx context.Context, id string, expectSeq int64, leaseUntil, now time.Time) (run.Item, error) {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), claimItem,
		[]any{formatTime(leaseUntil), formatTime(now), id, expectSeq}, nil, "claim the run item")
	if err != nil {
		return run.Item{}, err
	}
	if affected == 0 {
		return run.Item{}, errNotClaimed
	}
	return r.Get(ctx, id)
}

func (r *RunItemRepo) Persist(ctx context.Context, item run.Item, expectSeq int64) (bool, error) {
	checkpoint, err := item.Checkpoint.Encode()
	if err != nil {
		return false, err
	}

	affected, err := execWrite(ctx, r.store.writeFrom(ctx), persistItem, []any{
		string(item.Status), item.CurrentStep, item.Attempts, checkpoint, nullTime(item.LeaseUntil),
		nullTime(item.WakeAt), string(item.PauseReason), item.Error, formatTime(item.UpdatedAt),
		nullTime(item.FinishedAt), item.ID, expectSeq,
	}, nil, "persist the run item")
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *RunItemRepo) ByRun(ctx context.Context, runID string) ([]run.Item, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectItemsByRun, []any{runID}, scanItem, "list the run items")
}

func (r *RunItemRepo) ByTarget(ctx context.Context, targetID string, limit int) ([]run.Item, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectItemsByTarget, []any{targetID, limit}, scanItem,
		"list the run items of the target")
}

func (r *RunItemRepo) Due(ctx context.Context, now time.Time, limit int) ([]run.Item, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectDueItems, []any{formatTime(now), limit}, scanItem,
		"list the run items that are due to wake")
}

func (r *RunItemRepo) Stalled(ctx context.Context, now time.Time, limit int) ([]run.Item, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectStalledItems, []any{formatTime(now), limit}, scanItem,
		"list the run items whose lease expired")
}

func (r *RunItemRepo) Runnable(ctx context.Context, now time.Time, limit int) ([]run.Item, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectRunnableItems, []any{formatTime(now), limit}, scanItem,
		"list the runnable run items")
}

func (r *RunItemRepo) Requeue(ctx context.Context, id string, expectSeq int64, from run.Status, now time.Time) (bool, error) {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), requeueItem,
		[]any{formatTime(now), id, expectSeq, string(from)}, nil, "requeue the run item")
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *RunItemRepo) Counts(ctx context.Context, runID string) (map[run.Status]int, error) {
	rows, err := selectAll(ctx, r.store.execFrom(ctx), countItemsByStatus, []any{runID}, scanStatusCount,
		"count the run items by status")
	if err != nil {
		return nil, err
	}

	counts := make(map[run.Status]int, len(rows))
	for _, row := range rows {
		counts[row.status] = row.total
	}
	return counts, nil
}

func (r *RunItemRepo) StopAll(ctx context.Context, runID string, from []run.Status, to run.Status, reason run.PauseReason, now time.Time) (int64, error) {
	if len(from) == 0 {
		return 0, nil
	}

	finished := any(nil)
	if to.Terminal() {
		finished = formatTime(now)
	}

	args := []any{string(to), string(reason), formatTime(now), finished, runID}
	placeholders := make([]string, 0, len(from))
	for _, status := range from {
		placeholders = append(placeholders, "?")
		args = append(args, string(status))
	}

	query := stopItemsOfRun + "(" + strings.Join(placeholders, ", ") + ")"
	return execWrite(ctx, r.store.writeFrom(ctx), query, args, nil, "stop the run items")
}

func (r *RunItemRepo) ResumeAll(ctx context.Context, runID string, now time.Time) (int64, error) {
	return execWrite(ctx, r.store.writeFrom(ctx), resumeItemsOfRun, []any{formatTime(now), runID}, nil,
		"resume the run items")
}

func (r *RunItemRepo) List(ctx context.Context, q run.ItemQuery, page paging.Request) (paging.List[run.Item], error) {
	builder := squirrel.Select(itemColumns).From("run_items")
	if q.RunID != "" {
		builder = builder.Where(squirrel.Eq{"run_id": q.RunID})
	}
	if q.Status != nil {
		builder = builder.Where(squirrel.Eq{"status": string(*q.Status)})
	}

	keyset := itemKeyset(q)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[run.Item]{}, err
	}
	query, args, err := buildQuery(keyed, "run items")
	if err != nil {
		return paging.List[run.Item]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanItem, "list the run items")
	if err != nil {
		return paging.List[run.Item]{}, err
	}
	return keyset.Cut(rows, page)
}

func itemKeyset(q run.ItemQuery) paging.Keyset[run.Item] {
	return paging.Keyset[run.Item]{
		IDColumn: "id",
		ID:       func(i run.Item) string { return i.ID },
		Keys: []paging.SortKey[run.Item]{
			paging.TimeKey[run.Item]("createdAt", "created_at", func(i run.Item) any { return i.CreatedAt }),
		},
		Desc: q.Desc,
	}
}

type statusCount struct {
	status run.Status
	total  int
}

func scanStatusCount(rows *sql.Rows) (statusCount, error) {
	var (
		status string
		total  int
	)
	if err := rows.Scan(&status, &total); err != nil {
		return statusCount{}, err
	}
	return statusCount{status: run.Status(status), total: total}, nil
}

func scanItem(rows *sql.Rows) (run.Item, error) {
	var (
		item                 run.Item
		status, checkpoint   string
		pauseReason          string
		leaseUntil, wakeAt   sql.NullString
		createdAt, updatedAt string
		finishedAt           sql.NullString
	)
	if err := rows.Scan(
		&item.ID, &item.RunID, &item.SiteID, &item.TargetID, &status, &item.CurrentStep, &item.Attempts, &item.AdvanceSeq,
		&checkpoint, &leaseUntil, &wakeAt, &pauseReason, &item.Error, &createdAt, &updatedAt, &finishedAt,
	); err != nil {
		return run.Item{}, err
	}

	item.Status = run.Status(status)
	item.PauseReason = run.PauseReason(pauseReason)

	var err error
	if item.Checkpoint, err = run.DecodeCheckpoint(checkpoint); err != nil {
		return run.Item{}, err
	}
	if item.LeaseUntil, err = parseNullTime(leaseUntil); err != nil {
		return run.Item{}, err
	}
	if item.WakeAt, err = parseNullTime(wakeAt); err != nil {
		return run.Item{}, err
	}
	if item.CreatedAt, err = parseTime(createdAt); err != nil {
		return run.Item{}, err
	}
	if item.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return run.Item{}, err
	}
	if item.FinishedAt, err = parseNullTime(finishedAt); err != nil {
		return run.Item{}, err
	}
	return item, nil
}
