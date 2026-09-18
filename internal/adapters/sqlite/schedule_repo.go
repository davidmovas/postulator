package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/schedule"
	"github.com/davidmovas/postulator/internal/domain/template"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	scheduleColumns = `id, site_id, name, cron, interval_seconds, target_entity_id, target_status, target_limit,
		template_id, recipe, publish_mode, budget_max_usd, budget_max_tokens, enabled, next_run_at, last_run_id,
		created_by, created_at, updated_at`
	insertSchedule = `INSERT INTO schedules (` + scheduleColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updateSchedule = `UPDATE schedules SET name = ?, cron = ?, interval_seconds = ?, target_entity_id = ?,
		target_status = ?, target_limit = ?, template_id = ?, recipe = ?, publish_mode = ?, budget_max_usd = ?,
		budget_max_tokens = ?, enabled = ?, next_run_at = ?, last_run_id = ?, updated_at = ? WHERE id = ?`
	deleteSchedule     = `DELETE FROM schedules WHERE id = ?`
	selectSchedule     = `SELECT ` + scheduleColumns + ` FROM schedules WHERE id = ?`
	selectDueSchedules = `SELECT ` + scheduleColumns + ` FROM schedules
		WHERE enabled = 1 AND next_run_at IS NOT NULL AND next_run_at <= ? ORDER BY next_run_at, id LIMIT ?`
)

type ScheduleRepo struct {
	store *Store
}

func NewScheduleRepo(store *Store) *ScheduleRepo {
	return &ScheduleRepo{store: store}
}

func scheduleNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "schedule not found").WithDetail("scheduleId", id)
}

func scheduleConflict(name string) *errors.Error {
	return errors.New(errors.Conflict, "a schedule with this name already exists in the site").
		WithDetail("name", name)
}

func (r *ScheduleRepo) Insert(ctx context.Context, s schedule.Schedule) error {
	recipe, err := encodeJSON(s.Recipe)
	if err != nil {
		return err
	}

	_, err = execWrite(ctx, r.store.writeFrom(ctx), insertSchedule, []any{
		s.ID, s.SiteID, s.Name, s.Cron, seconds(s.Interval), nullString(s.Query.EntityID), s.Query.Status,
		s.Query.Limit, s.TemplateID, recipe, string(s.PublishMode), s.Budget.MaxUSD, s.Budget.MaxTokens,
		boolInt(s.Enabled), nullTime(s.NextRunAt), nullString(s.LastRunID), string(s.CreatedBy),
		formatTime(s.CreatedAt), formatTime(s.UpdatedAt),
	}, scheduleConflict(s.Name), "insert the schedule")
	return err
}

func (r *ScheduleRepo) Update(ctx context.Context, s schedule.Schedule) error {
	recipe, err := encodeJSON(s.Recipe)
	if err != nil {
		return err
	}

	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateSchedule, []any{
		s.Name, s.Cron, seconds(s.Interval), nullString(s.Query.EntityID), s.Query.Status, s.Query.Limit,
		s.TemplateID, recipe, string(s.PublishMode), s.Budget.MaxUSD, s.Budget.MaxTokens, boolInt(s.Enabled),
		nullTime(s.NextRunAt), nullString(s.LastRunID), formatTime(s.UpdatedAt), s.ID,
	}, scheduleConflict(s.Name), "update the schedule")
	return requireAffected(affected, err, scheduleNotFound(s.ID))
}

func (r *ScheduleRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteSchedule, []any{id}, nil, "delete the schedule")
	return requireAffected(affected, err, scheduleNotFound(id))
}

func (r *ScheduleRepo) Get(ctx context.Context, id string) (schedule.Schedule, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectSchedule, []any{id}, scanSchedule, scheduleNotFound(id),
		"read the schedule")
}

func (r *ScheduleRepo) Due(ctx context.Context, now time.Time, limit int) ([]schedule.Schedule, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectDueSchedules, []any{formatTime(now), limit}, scanSchedule,
		"list the schedules that are due")
}

func (r *ScheduleRepo) List(ctx context.Context, q schedule.Query, page paging.Request) (paging.List[schedule.Schedule], error) {
	builder := squirrel.Select(scheduleColumns).From("schedules")
	if q.SiteID != "" {
		builder = builder.Where(squirrel.Eq{"site_id": q.SiteID})
	}
	if q.Enabled != nil {
		builder = builder.Where(squirrel.Eq{"enabled": boolInt(*q.Enabled)})
	}

	keyset := scheduleKeyset(q.Desc)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[schedule.Schedule]{}, err
	}
	query, args, err := buildQuery(keyed, "schedules")
	if err != nil {
		return paging.List[schedule.Schedule]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanSchedule, "list the schedules")
	if err != nil {
		return paging.List[schedule.Schedule]{}, err
	}
	return keyset.Cut(rows, page)
}

func scheduleKeyset(desc bool) paging.Keyset[schedule.Schedule] {
	return paging.Keyset[schedule.Schedule]{
		IDColumn: "id",
		ID:       func(s schedule.Schedule) string { return s.ID },
		Keys: []paging.SortKey[schedule.Schedule]{
			paging.TimeKey[schedule.Schedule]("createdAt", "created_at", func(s schedule.Schedule) any { return s.CreatedAt }),
		},
		Desc: desc,
	}
}

func seconds(interval *time.Duration) int64 {
	if interval == nil {
		return 0
	}
	return int64(interval.Seconds())
}

func scanSchedule(rows *sql.Rows) (schedule.Schedule, error) {
	var (
		s                    schedule.Schedule
		intervalSeconds      int64
		entityID, lastRunID  sql.NullString
		nextRunAt            sql.NullString
		publishMode, creator string
		recipe               string
		enabled              int64
		createdAt, updatedAt string
	)
	if err := rows.Scan(&s.ID, &s.SiteID, &s.Name, &s.Cron, &intervalSeconds, &entityID, &s.Query.Status,
		&s.Query.Limit, &s.TemplateID, &recipe, &publishMode, &s.Budget.MaxUSD, &s.Budget.MaxTokens, &enabled,
		&nextRunAt, &lastRunID, &creator, &createdAt, &updatedAt); err != nil {
		return schedule.Schedule{}, err
	}

	if intervalSeconds > 0 {
		interval := time.Duration(intervalSeconds) * time.Second
		s.Interval = &interval
	}
	s.Query.EntityID = optString(entityID)
	s.LastRunID = optString(lastRunID)
	s.PublishMode = run.PublishMode(publishMode)
	s.CreatedBy = kctx.Actor(creator)
	s.Enabled = enabled == 1
	s.Recipe = []template.StepSpec{}
	if err := decodeJSON(recipe, &s.Recipe, "decode the schedule recipe"); err != nil {
		return schedule.Schedule{}, err
	}

	var err error
	if s.NextRunAt, err = parseNullTime(nextRunAt); err != nil {
		return schedule.Schedule{}, err
	}
	if s.CreatedAt, err = parseTime(createdAt); err != nil {
		return schedule.Schedule{}, err
	}
	if s.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return schedule.Schedule{}, err
	}
	return s, nil
}
