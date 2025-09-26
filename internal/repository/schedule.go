package repository

import (
	"Postulator/internal/models"
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
)

func (r *Repository) GetSchedules(ctx context.Context, limit int, offset int) (*models.PaginationResult[*models.Schedule], error) {
	query, args := builder.
		Select(
			"id",
			"site_id",
			"cron_expr",
			"is_active",
			"last_run",
			"next_run",
			"created_at",
			"updated_at",
		).
		From("schedules").
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedules: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var schedules []*models.Schedule
	for rows.Next() {
		var schedule models.Schedule
		if err = rows.Scan(
			&schedule.ID,
			&schedule.SiteID,
			&schedule.CronExpr,
			&schedule.IsActive,
			&schedule.LastRun,
			&schedule.NextRun,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, &schedule)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(id)").
		From("schedules").
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count schedules: %w", err)
	}

	return &models.PaginationResult[*models.Schedule]{
		Data:   schedules,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (r *Repository) GetSchedule(ctx context.Context, id int64) (*models.Schedule, error) {
	query, args := builder.
		Select(
			"id",
			"site_id",
			"cron_expr",
			"is_active",
			"last_run",
			"next_run",
			"created_at",
			"updated_at",
		).
		From("schedules").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var schedule models.Schedule
	if err := r.db.QueryRowContext(ctx, query, args...).
		Scan(
			&schedule.ID,
			&schedule.SiteID,
			&schedule.CronExpr,
			&schedule.IsActive,
			&schedule.LastRun,
			&schedule.NextRun,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		); err != nil {
		return nil, fmt.Errorf("failed to query schedule: %w", err)
	}

	return &schedule, nil
}

func (r *Repository) GetSchedulesBySiteID(ctx context.Context, siteID int64, limit int, offset int) (*models.PaginationResult[*models.Schedule], error) {
	query, args := builder.
		Select(
			"id",
			"site_id",
			"cron_expr",
			"is_active",
			"last_run",
			"next_run",
			"created_at",
			"updated_at",
		).
		From("schedules").
		Where(squirrel.Eq{"site_id": siteID}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedules by site: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var schedules []*models.Schedule
	for rows.Next() {
		var schedule models.Schedule
		if err = rows.Scan(
			&schedule.ID,
			&schedule.SiteID,
			&schedule.CronExpr,
			&schedule.IsActive,
			&schedule.LastRun,
			&schedule.NextRun,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, &schedule)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(id)").
		From("schedules").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count schedules by site: %w", err)
	}

	return &models.PaginationResult[*models.Schedule]{
		Data:   schedules,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (r *Repository) CreateSchedule(ctx context.Context, schedule *models.Schedule) (*models.Schedule, error) {
	query, args := builder.
		Insert("schedules").
		Columns(
			"site_id",
			"cron_expr",
			"is_active",
			"last_run",
			"next_run",
			"created_at",
			"updated_at",
		).
		Values(
			schedule.SiteID,
			schedule.CronExpr,
			schedule.IsActive,
			schedule.LastRun,
			schedule.NextRun,
			schedule.CreatedAt,
			schedule.UpdatedAt,
		).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get insert id: %w", err)
	}

	schedule.ID = id
	return schedule, nil
}

func (r *Repository) UpdateSchedule(ctx context.Context, schedule *models.Schedule) (*models.Schedule, error) {
	query, args := builder.
		Update("schedules").
		Set("site_id", schedule.SiteID).
		Set("cron_expr", schedule.CronExpr).
		Set("is_active", schedule.IsActive).
		Set("last_run", schedule.LastRun).
		Set("next_run", schedule.NextRun).
		Set("updated_at", schedule.UpdatedAt).
		Where(squirrel.Eq{"id": schedule.ID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	return schedule, nil
}

func (r *Repository) DeleteSchedule(ctx context.Context, id int64) error {
	query, args := builder.
		Delete("schedules").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	return nil
}

func (r *Repository) GetActive(ctx context.Context) ([]*models.Schedule, error) {
	query, args := builder.
		Select(
			"id",
			"site_id",
			"cron_expr",
			"is_active",
			"last_run",
			"next_run",
			"created_at",
			"updated_at",
		).
		From("schedules").
		Where(squirrel.Eq{"is_active": true}).
		OrderBy("created_at ASC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query active schedules: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var schedules []*models.Schedule
	for rows.Next() {
		var schedule models.Schedule
		if err = rows.Scan(
			&schedule.ID,
			&schedule.SiteID,
			&schedule.CronExpr,
			&schedule.IsActive,
			&schedule.LastRun,
			&schedule.NextRun,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*models.Schedule, error) {
	return r.GetSchedule(ctx, id)
}

func (r *Repository) UpdateLastRun(ctx context.Context, id int64) error {
	query, args := builder.
		Update("schedules").
		Set("last_run", time.Now()).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update last run: %w", err)
	}

	return nil
}

func (r *Repository) UpdateNextRun(ctx context.Context, id int64, nextRun int64) error {
	nextRunTime := time.Unix(nextRun, 0)

	query, args := builder.
		Update("schedules").
		Set("next_run", nextRunTime).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update next run: %w", err)
	}

	return nil
}
