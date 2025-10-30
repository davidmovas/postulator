package job

import (
	"Postulator/internal/infra/database"
	"Postulator/pkg/dbx"
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Masterminds/squirrel"
)

var _ IRepository = (*JobRepository)(nil)

type JobRepository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewJobRepository(c di.Container) (*JobRepository, error) {
	var db *database.DB
	if err := c.Resolve(&db); err != nil {
		return nil, err
	}

	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	return &JobRepository{
		db:     db,
		logger: l,
	}, nil
}

func (r *JobRepository) Create(ctx context.Context, job *Job) error {
	// prepare JSON fields
	var weekdaysStr, monthdaysStr *string
	if len(job.Weekdays) > 0 {
		if b, err := json.Marshal(job.Weekdays); err == nil {
			s := string(b)
			weekdaysStr = &s
		}
	}

	query, args := dbx.ST.
		Insert("jobs").
		Columns(
			"name",
			"site_id",
			"category_id",
			"prompt_id",
			"ai_provider_id",
			"ai_model",
			"requires_validation",
			"schedule_type",
			"interval_value",
			"interval_unit",
			"schedule_hour",
			"schedule_minute",
			"weekdays",
			"monthdays",
			"jitter_enabled",
			"jitter_minutes",
			"status",
			"next_run_at",
		).
		Values(
			job.Name,
			job.SiteID,
			job.CategoryID,
			job.PromptID,
			job.AIProviderID,
			job.AIModel,
			job.RequiresValidation,
			job.ScheduleType,
			job.IntervalValue,
			job.IntervalUnit,
			job.ScheduleHour,
			job.ScheduleMinute,
			weekdaysStr,
			monthdaysStr,
			job.JitterEnabled,
			job.JitterMinutes,
			job.Status,
			job.NextRunAt,
		).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Internal(err)
	}

	return nil
}

func (r *JobRepository) GetByID(ctx context.Context, id int64) (*Job, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"name",
			"site_id",
			"category_id",
			"prompt_id",
			"ai_provider_id",
			"ai_model",
			"requires_validation",
			"schedule_type",
			"interval_value",
			"interval_unit",
			"schedule_hour",
			"schedule_minute",
			"weekdays",
			"monthdays",
			"jitter_enabled",
			"jitter_minutes",
			"status",
			"last_run_at",
			"next_run_at",
			"created_at",
			"updated_at",
		).
		From("jobs").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var job Job
	var intervalValue sql.NullInt64
	var intervalUnit sql.NullString
	var scheduleHour sql.NullInt64
	var scheduleMinute sql.NullInt64
	var weekdaysStr sql.NullString
	var monthdaysStr sql.NullString

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&job.ID,
		&job.Name,
		&job.SiteID,
		&job.CategoryID,
		&job.PromptID,
		&job.AIProviderID,
		&job.AIModel,
		&job.RequiresValidation,
		&job.ScheduleType,
		&intervalValue,
		&intervalUnit,
		&scheduleHour,
		&scheduleMinute,
		&weekdaysStr,
		&monthdaysStr,
		&job.JitterEnabled,
		&job.JitterMinutes,
		&job.Status,
		&job.LastRunAt,
		&job.NextRunAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("job", id)
	case err != nil:
		return nil, errors.Internal(err)
	}

	if intervalValue.Valid {
		v := int(intervalValue.Int64)
		job.IntervalValue = &v
	}
	if intervalUnit.Valid {
		v := intervalUnit.String
		unit := IntervalUnit(v)
		job.IntervalUnit = &unit
	}
	if scheduleHour.Valid {
		v := int(scheduleHour.Int64)
		job.ScheduleHour = &v
	}
	if scheduleMinute.Valid {
		v := int(scheduleMinute.Int64)
		job.ScheduleMinute = &v
	}
	if weekdaysStr.Valid {
		var arr []int
		_ = json.Unmarshal([]byte(weekdaysStr.String), &arr)
		job.Weekdays = arr
	}

	return &job, nil
}

func (r *JobRepository) GetAll(ctx context.Context) ([]*Job, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"name",
			"site_id",
			"category_id",
			"prompt_id",
			"ai_provider_id",
			"ai_model",
			"requires_validation",
			"schedule_type",
			"interval_value",
			"interval_unit",
			"schedule_hour",
			"schedule_minute",
			"weekdays",
			"monthdays",
			"jitter_enabled",
			"jitter_minutes",
			"status",
			"last_run_at",
			"next_run_at",
			"created_at",
			"updated_at",
		).
		From("jobs").
		OrderBy("created_at DESC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Internal(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var jobs []*Job
	for rows.Next() {
		var job Job
		var intervalValue sql.NullInt64
		var intervalUnit sql.NullString
		var scheduleHour sql.NullInt64
		var scheduleMinute sql.NullInt64
		var weekdaysStr sql.NullString
		var monthdaysStr sql.NullString
		if err = rows.Scan(
			&job.ID,
			&job.Name,
			&job.SiteID,
			&job.CategoryID,
			&job.PromptID,
			&job.AIProviderID,
			&job.AIModel,
			&job.RequiresValidation,
			&job.ScheduleType,
			&intervalValue,
			&intervalUnit,
			&scheduleHour,
			&scheduleMinute,
			&weekdaysStr,
			&monthdaysStr,
			&job.JitterEnabled,
			&job.JitterMinutes,
			&job.Status,
			&job.LastRunAt,
			&job.NextRunAt,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, errors.Internal(err)
		}
		if intervalValue.Valid {
			v := int(intervalValue.Int64)
			job.IntervalValue = &v
		}
		if intervalUnit.Valid {
			unit := IntervalUnit(intervalUnit.String)
			job.IntervalUnit = &unit
		}
		if scheduleHour.Valid {
			v := int(scheduleHour.Int64)
			job.ScheduleHour = &v
		}
		if scheduleMinute.Valid {
			v := int(scheduleMinute.Int64)
			job.ScheduleMinute = &v
		}
		if weekdaysStr.Valid {
			var arr []int
			_ = json.Unmarshal([]byte(weekdaysStr.String), &arr)
			job.Weekdays = arr
		}

		jobs = append(jobs, &job)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Internal(err)
	}

	return jobs, nil
}

func (r *JobRepository) GetActive(ctx context.Context) ([]*Job, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"name",
			"site_id",
			"category_id",
			"prompt_id",
			"ai_provider_id",
			"ai_model",
			"requires_validation",
			"schedule_type",
			"interval_value",
			"interval_unit",
			"schedule_hour",
			"schedule_minute",
			"weekdays",
			"monthdays",
			"jitter_enabled",
			"jitter_minutes",
			"status",
			"last_run_at",
			"next_run_at",
			"created_at",
			"updated_at",
		).
		From("jobs").
		Where(squirrel.Eq{"status": StatusActive}).
		OrderBy("next_run_at ASC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Internal(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var jobs []*Job
	for rows.Next() {
		var job Job
		var intervalValue sql.NullInt64
		var intervalUnit sql.NullString
		var scheduleHour sql.NullInt64
		var scheduleMinute sql.NullInt64
		var weekdaysStr sql.NullString
		var monthdaysStr sql.NullString
		if err = rows.Scan(
			&job.ID,
			&job.Name,
			&job.SiteID,
			&job.CategoryID,
			&job.PromptID,
			&job.AIProviderID,
			&job.AIModel,
			&job.RequiresValidation,
			&job.ScheduleType,
			&intervalValue,
			&intervalUnit,
			&scheduleHour,
			&scheduleMinute,
			&weekdaysStr,
			&monthdaysStr,
			&job.JitterEnabled,
			&job.JitterMinutes,
			&job.Status,
			&job.LastRunAt,
			&job.NextRunAt,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, errors.Internal(err)
		}
		if intervalValue.Valid {
			v := int(intervalValue.Int64)
			job.IntervalValue = &v
		}
		if intervalUnit.Valid {
			unit := IntervalUnit(intervalUnit.String)
			job.IntervalUnit = &unit
		}
		if scheduleHour.Valid {
			v := int(scheduleHour.Int64)
			job.ScheduleHour = &v
		}
		if scheduleMinute.Valid {
			v := int(scheduleMinute.Int64)
			job.ScheduleMinute = &v
		}
		if weekdaysStr.Valid {
			var arr []int
			_ = json.Unmarshal([]byte(weekdaysStr.String), &arr)
			job.Weekdays = arr
		}

		jobs = append(jobs, &job)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Internal(err)
	}

	return jobs, nil
}

func (r *JobRepository) Update(ctx context.Context, job *Job) error {
	var weekdaysStr, monthdaysStr *string
	if len(job.Weekdays) > 0 {
		if b, err := json.Marshal(job.Weekdays); err == nil {
			s := string(b)
			weekdaysStr = &s
		}
	}

	query, args := dbx.ST.
		Update("jobs").
		Set("name", job.Name).
		Set("site_id", job.SiteID).
		Set("category_id", job.CategoryID).
		Set("prompt_id", job.PromptID).
		Set("ai_provider_id", job.AIProviderID).
		Set("ai_model", job.AIModel).
		Set("requires_validation", job.RequiresValidation).
		Set("schedule_type", job.ScheduleType).
		Set("interval_value", job.IntervalValue).
		Set("interval_unit", job.IntervalUnit).
		Set("schedule_hour", job.ScheduleHour).
		Set("schedule_minute", job.ScheduleMinute).
		Set("weekdays", weekdaysStr).
		Set("monthdays", monthdaysStr).
		Set("jitter_enabled", job.JitterEnabled).
		Set("jitter_minutes", job.JitterMinutes).
		Set("status", job.Status).
		Set("last_run_at", job.LastRunAt).
		Set("next_run_at", job.NextRunAt).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": job.ID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsNoRows(err):
		return errors.NotFound("job", job.ID)
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}

func (r *JobRepository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("jobs").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Internal(err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return errors.Internal(err)
	}

	if affected == 0 {
		return errors.NotFound("job", id)
	}

	return nil
}

func (r *JobRepository) GetDueJobs(ctx context.Context, now time.Time) ([]*Job, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"name",
			"site_id",
			"category_id",
			"prompt_id",
			"ai_provider_id",
			"ai_model",
			"requires_validation",
			"schedule_type",
			"interval_value",
			"interval_unit",
			"schedule_hour",
			"schedule_minute",
			"weekdays",
			"monthdays",
			"jitter_enabled",
			"jitter_minutes",
			"status",
			"last_run_at",
			"next_run_at",
			"created_at",
			"updated_at",
		).
		From("jobs").
		Where(squirrel.Eq{"status": StatusActive}).
		Where(squirrel.LtOrEq{"next_run_at": now}).
		OrderBy("next_run_at ASC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Internal(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var jobs []*Job
	for rows.Next() {
		var job Job
		var intervalValue sql.NullInt64
		var intervalUnit sql.NullString
		var scheduleHour sql.NullInt64
		var scheduleMinute sql.NullInt64
		var weekdaysStr sql.NullString
		var monthdaysStr sql.NullString
		if err = rows.Scan(
			&job.ID,
			&job.Name,
			&job.SiteID,
			&job.CategoryID,
			&job.PromptID,
			&job.AIProviderID,
			&job.AIModel,
			&job.RequiresValidation,
			&job.ScheduleType,
			&intervalValue,
			&intervalUnit,
			&scheduleHour,
			&scheduleMinute,
			&weekdaysStr,
			&monthdaysStr,
			&job.JitterEnabled,
			&job.JitterMinutes,
			&job.Status,
			&job.LastRunAt,
			&job.NextRunAt,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, errors.Internal(err)
		}
		if intervalValue.Valid {
			v := int(intervalValue.Int64)
			job.IntervalValue = &v
		}
		if intervalUnit.Valid {
			unit := IntervalUnit(intervalUnit.String)
			job.IntervalUnit = &unit
		}
		if scheduleHour.Valid {
			v := int(scheduleHour.Int64)
			job.ScheduleHour = &v
		}
		if scheduleMinute.Valid {
			v := int(scheduleMinute.Int64)
			job.ScheduleMinute = &v
		}
		if weekdaysStr.Valid {
			var arr []int
			_ = json.Unmarshal([]byte(weekdaysStr.String), &arr)
			job.Weekdays = arr
		}

		jobs = append(jobs, &job)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Internal(err)
	}

	return jobs, nil
}
