package repository

import (
	"Postulator/internal/models"
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
)

func (r *Repository) GetJobs(ctx context.Context, limit int, offset int) (*models.PaginationResult[*models.PostingJob], error) {
	query, args := builder.
		Select(
			"id",
			"type",
			"site_id",
			"article_id",
			"status",
			"progress",
			"error_msg",
			"started_at",
			"completed_at",
			"created_at",
		).
		From("posting_jobs").
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var jobs []*models.PostingJob
	for rows.Next() {
		var job models.PostingJob
		if err = rows.Scan(
			&job.ID,
			&job.Type,
			&job.SiteID,
			&job.ArticleID,
			&job.Status,
			&job.Progress,
			&job.ErrorMsg,
			&job.StartedAt,
			&job.CompletedAt,
			&job.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, &job)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(*)").
		From("posting_jobs").
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count jobs: %w", err)
	}

	return &models.PaginationResult[*models.PostingJob]{
		Data:   jobs,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (r *Repository) GetJob(ctx context.Context, id int64) (*models.PostingJob, error) {
	query, args := builder.
		Select(
			"id",
			"type",
			"site_id",
			"article_id",
			"status",
			"progress",
			"error_msg",
			"started_at",
			"completed_at",
			"created_at",
		).
		From("posting_jobs").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var job models.PostingJob
	if err := r.db.QueryRowContext(ctx, query, args...).
		Scan(
			&job.ID,
			&job.Type,
			&job.SiteID,
			&job.ArticleID,
			&job.Status,
			&job.Progress,
			&job.ErrorMsg,
			&job.StartedAt,
			&job.CompletedAt,
			&job.CreatedAt,
		); err != nil {
		return nil, fmt.Errorf("failed to query job: %w", err)
	}

	return &job, nil
}

func (r *Repository) GetJobsBySiteID(ctx context.Context, siteID int64, limit int, offset int) (*models.PaginationResult[*models.PostingJob], error) {
	query, args := builder.
		Select(
			"id",
			"type",
			"site_id",
			"article_id",
			"status",
			"progress",
			"error_msg",
			"started_at",
			"completed_at",
			"created_at",
		).
		From("posting_jobs").
		Where(squirrel.Eq{"site_id": siteID}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs by site: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var jobs []*models.PostingJob
	for rows.Next() {
		var job models.PostingJob
		if err = rows.Scan(
			&job.ID,
			&job.Type,
			&job.SiteID,
			&job.ArticleID,
			&job.Status,
			&job.Progress,
			&job.ErrorMsg,
			&job.StartedAt,
			&job.CompletedAt,
			&job.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, &job)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(*)").
		From("posting_jobs").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count jobs by site: %w", err)
	}

	return &models.PaginationResult[*models.PostingJob]{
		Data:   jobs,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (r *Repository) CreateJob(ctx context.Context, job *models.PostingJob) (*models.PostingJob, error) {
	query, args := builder.
		Insert("posting_jobs").
		Columns(
			"type",
			"site_id",
			"article_id",
			"status",
			"progress",
			"error_msg",
			"started_at",
			"completed_at",
			"created_at",
		).
		Values(
			job.Type,
			job.SiteID,
			job.ArticleID,
			job.Status,
			job.Progress,
			job.ErrorMsg,
			job.StartedAt,
			job.CompletedAt,
			job.CreatedAt,
		).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get insert id: %w", err)
	}

	job.ID = id
	return job, nil
}

func (r *Repository) UpdateJob(ctx context.Context, job *models.PostingJob) (*models.PostingJob, error) {
	query, args := builder.
		Update("posting_jobs").
		Set("type", job.Type).
		Set("site_id", job.SiteID).
		Set("article_id", job.ArticleID).
		Set("status", job.Status).
		Set("progress", job.Progress).
		Set("error_msg", job.ErrorMsg).
		Set("started_at", job.StartedAt).
		Set("completed_at", job.CompletedAt).
		Where(squirrel.Eq{"id": job.ID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	return job, nil
}

func (r *Repository) DeleteJob(ctx context.Context, id int64) error {
	query, args := builder.
		Delete("posting_jobs").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	return nil
}

func (r *Repository) GetJobsByStatus(ctx context.Context, status string, limit int, offset int) (*models.PaginationResult[*models.PostingJob], error) {
	query, args := builder.
		Select(
			"id",
			"type",
			"site_id",
			"article_id",
			"status",
			"progress",
			"error_msg",
			"started_at",
			"completed_at",
			"created_at",
		).
		From("posting_jobs").
		Where(squirrel.Eq{"status": status}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs by status: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var jobs []*models.PostingJob
	for rows.Next() {
		var job models.PostingJob
		if err = rows.Scan(
			&job.ID,
			&job.Type,
			&job.SiteID,
			&job.ArticleID,
			&job.Status,
			&job.Progress,
			&job.ErrorMsg,
			&job.StartedAt,
			&job.CompletedAt,
			&job.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, &job)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(*)").
		From("posting_jobs").
		Where(squirrel.Eq{"status": status}).
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count jobs by status: %w", err)
	}

	return &models.PaginationResult[*models.PostingJob]{
		Data:   jobs,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}
