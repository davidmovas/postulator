package job

import (
	"Postulator/internal/infra/database"
	"Postulator/pkg/dbx"
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"

	"github.com/Masterminds/squirrel"
)

var _ IExecutionRepository = (*ExecRepository)(nil)

type ExecRepository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewExecutionRepository(c di.Container) (*ExecRepository, error) {
	var db *database.DB
	if err := c.Resolve(&db); err != nil {
		return nil, err
	}

	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	return &ExecRepository{
		db:     db,
		logger: l,
	}, nil
}

func (r *ExecRepository) Create(ctx context.Context, exec *Execution) error {
	query, args := dbx.ST.
		Insert("job_executions").
		Columns(
			"job_id",
			"topic_id",
			"generated_title",
			"generated_content",
			"status",
			"error_message",
			"article_id",
			"started_at",
			"generated_at",
			"validated_at",
			"published_at",
		).
		Values(
			exec.JobID,
			exec.TopicID,
			exec.GeneratedTitle,
			exec.GeneratedContent,
			exec.Status,
			exec.ErrorMessage,
			exec.ArticleID,
			exec.StartedAt,
			exec.GeneratedAt,
			exec.ValidatedAt,
			exec.PublishedAt,
		).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Internal(err)
	}

	return nil
}

func (r *ExecRepository) GetByID(ctx context.Context, id int64) (*Execution, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"job_id",
			"topic_id",
			"generated_title",
			"generated_content",
			"status",
			"error_message",
			"article_id",
			"started_at",
			"generated_at",
			"validated_at",
			"published_at",
		).
		From("job_executions").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var exec Execution
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&exec.ID,
		&exec.JobID,
		&exec.TopicID,
		&exec.GeneratedTitle,
		&exec.GeneratedContent,
		&exec.Status,
		&exec.ErrorMessage,
		&exec.ArticleID,
		&exec.StartedAt,
		&exec.GeneratedAt,
		&exec.ValidatedAt,
		&exec.PublishedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("execution", id)
	case err != nil:
		return nil, errors.Internal(err)
	}

	return &exec, nil
}

func (r *ExecRepository) GetByJobID(ctx context.Context, jobID int64) ([]*Execution, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"job_id",
			"topic_id",
			"generated_title",
			"generated_content",
			"status",
			"error_message",
			"article_id",
			"started_at",
			"generated_at",
			"validated_at",
			"published_at",
		).
		From("job_executions").
		Where(squirrel.Eq{"job_id": jobID}).
		OrderBy("started_at DESC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Internal(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var executions []*Execution
	for rows.Next() {
		var exec Execution
		if err = rows.Scan(
			&exec.ID,
			&exec.JobID,
			&exec.TopicID,
			&exec.GeneratedTitle,
			&exec.GeneratedContent,
			&exec.Status,
			&exec.ErrorMessage,
			&exec.ArticleID,
			&exec.StartedAt,
			&exec.GeneratedAt,
			&exec.ValidatedAt,
			&exec.PublishedAt,
		); err != nil {
			return nil, errors.Internal(err)
		}

		executions = append(executions, &exec)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Internal(err)
	}

	return executions, nil
}

func (r *ExecRepository) GetPendingValidation(ctx context.Context) ([]*Execution, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"job_id",
			"topic_id",
			"generated_title",
			"generated_content",
			"status",
			"error_message",
			"article_id",
			"started_at",
			"generated_at",
			"validated_at",
			"published_at",
		).
		From("job_executions").
		Where(squirrel.Eq{"status": ExecutionPendingValidation}).
		OrderBy("started_at ASC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Internal(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var executions []*Execution
	for rows.Next() {
		var exec Execution
		if err = rows.Scan(
			&exec.ID,
			&exec.JobID,
			&exec.TopicID,
			&exec.GeneratedTitle,
			&exec.GeneratedContent,
			&exec.Status,
			&exec.ErrorMessage,
			&exec.ArticleID,
			&exec.StartedAt,
			&exec.GeneratedAt,
			&exec.ValidatedAt,
			&exec.PublishedAt,
		); err != nil {
			return nil, errors.Internal(err)
		}

		executions = append(executions, &exec)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Internal(err)
	}

	return executions, nil
}

func (r *ExecRepository) Update(ctx context.Context, exec *Execution) error {
	query, args := dbx.ST.
		Update("job_executions").
		Set("generated_title", exec.GeneratedTitle).
		Set("generated_content", exec.GeneratedContent).
		Set("status", exec.Status).
		Set("error_message", exec.ErrorMessage).
		Set("article_id", exec.ArticleID).
		Set("generated_at", exec.GeneratedAt).
		Set("validated_at", exec.ValidatedAt).
		Set("published_at", exec.PublishedAt).
		Where(squirrel.Eq{"id": exec.ID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsNoRows(err):
		return errors.NotFound("execution", exec.ID)
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}

func (r *ExecRepository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("job_executions").
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
		return errors.NotFound("execution", id)
	}

	return nil
}
