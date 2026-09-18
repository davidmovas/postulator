package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/run"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	runColumns = `id, site_id, kind, status, targets, recipe, template_id, template_version, publish_mode,
		budget_max_usd, budget_max_tokens, stats_items, stats_done, stats_failed, stats_tokens, stats_usd,
		created_by, parent_run_id, pause_reason, error, deadline_at, created_at, started_at, finished_at`
	insertRun = `INSERT INTO runs (` + runColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updateRun = `UPDATE runs SET status = ?, stats_items = ?, stats_done = ?, stats_failed = ?, stats_tokens = ?,
		stats_usd = ?, pause_reason = ?, error = ?, started_at = ?, finished_at = ? WHERE id = ?`
	selectRun       = `SELECT ` + runColumns + ` FROM runs WHERE id = ?`
	selectActiveRun = `SELECT ` + runColumns + ` FROM runs WHERE status IN ('pending', 'running', 'waiting')
		ORDER BY created_at, id`
	selectStaleRun = `SELECT ` + runColumns + ` FROM runs
		WHERE status IN ('pending', 'running', 'waiting', 'paused') AND deadline_at <= ? ORDER BY created_at, id LIMIT ?`
)

type RunRepo struct {
	store *Store
}

func NewRunRepo(store *Store) *RunRepo {
	return &RunRepo{store: store}
}

func runNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "run not found").WithDetail("runId", id)
}

func (r *RunRepo) Insert(ctx context.Context, record run.Run) error {
	targets, err := encodeJSON(record.Targets)
	if err != nil {
		return err
	}
	recipe, err := encodeJSON(record.Recipe)
	if err != nil {
		return err
	}

	_, err = execWrite(ctx, r.store.writeFrom(ctx), insertRun, []any{
		record.ID, record.SiteID, string(record.Kind), string(record.Status), targets, recipe,
		record.TemplateID, record.TemplateVersion, string(record.PublishMode),
		record.Budget.MaxUSD, record.Budget.MaxTokens,
		record.Stats.Items, record.Stats.Done, record.Stats.Failed, record.Stats.Tokens, record.Stats.USD,
		string(record.CreatedBy), nullString(record.ParentRunID), string(record.PauseReason), record.Error,
		formatTime(record.DeadlineAt), formatTime(record.CreatedAt),
		nullTime(record.StartedAt), nullTime(record.FinishedAt),
	}, errors.New(errors.Conflict, "a run with this id already exists"), "insert the run")
	return err
}

func (r *RunRepo) Update(ctx context.Context, record run.Run) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateRun, []any{
		string(record.Status), record.Stats.Items, record.Stats.Done, record.Stats.Failed,
		record.Stats.Tokens, record.Stats.USD, string(record.PauseReason), record.Error,
		nullTime(record.StartedAt), nullTime(record.FinishedAt), record.ID,
	}, nil, "update the run")
	return requireAffected(affected, err, runNotFound(record.ID))
}

func (r *RunRepo) Get(ctx context.Context, id string) (run.Run, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectRun, []any{id}, scanRun, runNotFound(id), "read the run")
}

func (r *RunRepo) Active(ctx context.Context) ([]run.Run, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectActiveRun, nil, scanRun, "list the active runs")
}

func (r *RunRepo) PastDeadline(ctx context.Context, now time.Time, limit int) ([]run.Run, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectStaleRun, []any{formatTime(now), limit}, scanRun,
		"list the runs past their deadline")
}

func runKeyset(q run.Query) paging.Keyset[run.Run] {
	keys := []paging.SortKey[run.Run]{}
	if q.Sort == run.SortStatus {
		keys = append(keys, paging.TextKey[run.Run]("status", "status", func(r run.Run) any { return string(r.Status) }))
	}
	keys = append(keys, paging.TimeKey[run.Run]("createdAt", "created_at", func(r run.Run) any { return r.CreatedAt }))

	return paging.Keyset[run.Run]{
		IDColumn: "id",
		ID:       func(r run.Run) string { return r.ID },
		Keys:     keys,
		Desc:     q.Desc,
	}
}

func (r *RunRepo) List(ctx context.Context, q run.Query, page paging.Request) (paging.List[run.Run], error) {
	builder := squirrel.Select(runColumns).From("runs")
	if q.SiteID != "" {
		builder = builder.Where(squirrel.Eq{"site_id": q.SiteID})
	}
	if q.Status != nil {
		builder = builder.Where(squirrel.Eq{"status": string(*q.Status)})
	}
	if q.Kind != nil {
		builder = builder.Where(squirrel.Eq{"kind": string(*q.Kind)})
	}

	keyset := runKeyset(q)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[run.Run]{}, err
	}
	query, args, err := buildQuery(keyed, "runs")
	if err != nil {
		return paging.List[run.Run]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanRun, "list the runs")
	if err != nil {
		return paging.List[run.Run]{}, err
	}
	return keyset.Cut(rows, page)
}

func scanRun(rows *sql.Rows) (run.Run, error) {
	var (
		record                             run.Run
		kind, status, publishMode, creator string
		targets, recipe                    string
		pauseReason                        string
		parentRunID                        sql.NullString
		deadlineAt, createdAt              string
		startedAt, finishedAt              sql.NullString
	)
	if err := rows.Scan(
		&record.ID, &record.SiteID, &kind, &status, &targets, &recipe, &record.TemplateID, &record.TemplateVersion,
		&publishMode, &record.Budget.MaxUSD, &record.Budget.MaxTokens, &record.Stats.Items, &record.Stats.Done,
		&record.Stats.Failed, &record.Stats.Tokens, &record.Stats.USD, &creator, &parentRunID, &pauseReason,
		&record.Error, &deadlineAt, &createdAt, &startedAt, &finishedAt,
	); err != nil {
		return run.Run{}, err
	}

	record.Kind = run.Kind(kind)
	record.Status = run.Status(status)
	record.PublishMode = run.PublishMode(publishMode)
	record.CreatedBy = kctx.Actor(creator)
	record.PauseReason = run.PauseReason(pauseReason)
	record.ParentRunID = optString(parentRunID)

	if err := decodeJSON(targets, &record.Targets, "stored run targets are not readable"); err != nil {
		return run.Run{}, err
	}
	if err := decodeJSON(recipe, &record.Recipe, "stored run recipe is not readable"); err != nil {
		return run.Run{}, err
	}

	var err error
	if record.DeadlineAt, err = parseTime(deadlineAt); err != nil {
		return run.Run{}, err
	}
	if record.CreatedAt, err = parseTime(createdAt); err != nil {
		return run.Run{}, err
	}
	if record.StartedAt, err = parseNullTime(startedAt); err != nil {
		return run.Run{}, err
	}
	if record.FinishedAt, err = parseNullTime(finishedAt); err != nil {
		return run.Run{}, err
	}
	return record, nil
}
