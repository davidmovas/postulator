package job

import (
	"Postulator/internal/infra/database"
	"Postulator/pkg/dbx"
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"
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
			"schedule_time",
			"schedule_day",
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
			job.ScheduleTime,
			job.ScheduleDay,
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
			"schedule_time",
			"schedule_day",
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
		&job.ScheduleTime,
		&job.ScheduleDay,
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
			"schedule_time",
			"schedule_day",
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
			&job.ScheduleTime,
			&job.ScheduleDay,
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
			"schedule_time",
			"schedule_day",
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
			&job.ScheduleTime,
			&job.ScheduleDay,
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

		jobs = append(jobs, &job)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Internal(err)
	}

	return jobs, nil
}

func (r *JobRepository) Update(ctx context.Context, job *Job) error {
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
		Set("schedule_time", job.ScheduleTime).
		Set("schedule_day", job.ScheduleDay).
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
			"schedule_time",
			"schedule_day",
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
			&job.ScheduleTime,
			&job.ScheduleDay,
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

		jobs = append(jobs, &job)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Internal(err)
	}

	return jobs, nil
}
