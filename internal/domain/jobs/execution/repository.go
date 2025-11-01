package execution

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/pkg/dbx"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Repository = (*repository)(nil)

type repository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewRepository(db *database.DB, logger *logger.Logger) Repository {
	return &repository{
		db: db,
		logger: logger.
			WithScope("repository").
			WithScope("job_executions"),
	}
}

func (r *repository) Create(ctx context.Context, exec *Execution) error {
	query, args := dbx.ST.
		Insert("job_executions").
		Columns(
			"job_id", "topic_id", "article_id",
			"prompt_id", "ai_provider_id", "ai_model", "category_id",
			"status", "error_message",
			"generation_time_ms", "tokens_used",
			"started_at", "generated_at", "validated_at", "published_at", "completed_at",
		).
		Values(
			exec.JobID, exec.TopicID, exec.ArticleID,
			exec.PromptID, exec.AIProviderID, exec.AIModel, exec.CategoryID,
			exec.Status, exec.ErrorMessage,
			exec.GenerationTimeMs, exec.TokensUsed,
			exec.StartedAt, exec.GeneratedAt, exec.ValidatedAt, exec.PublishedAt, exec.CompletedAt,
		).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsForeignKeyViolation(err):
		return errors.Validation("Invalid job, topic, article, prompt, AI provider or category ID")
	case err != nil:
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}

	exec.ID = id
	return nil
}

func (r *repository) GetByID(ctx context.Context, id int64) (*Execution, error) {
	query, args := dbx.ST.
		Select(
			"id", "job_id", "topic_id", "article_id",
			"prompt_id", "ai_provider_id", "ai_model", "category_id",
			"status", "error_message",
			"generation_time_ms", "tokens_used",
			"started_at", "generated_at", "validated_at", "published_at", "completed_at",
		).
		From("job_executions").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	exec, err := r.scanExecution(query, args, ctx)
	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("job_execution", id)
	case err != nil:
		return nil, errors.Database(err)
	}

	return exec, nil
}

func (r *repository) GetByJobID(ctx context.Context, jobID int64, limit, offset int) ([]*Execution, int, error) {
	query, args := dbx.ST.
		Select(
			"id", "job_id", "topic_id", "article_id",
			"prompt_id", "ai_provider_id", "ai_model", "category_id",
			"status", "error_message",
			"generation_time_ms", "tokens_used",
			"started_at", "generated_at", "validated_at", "published_at", "completed_at",
		).
		From("job_executions").
		Where(squirrel.Eq{"job_id": jobID}).
		OrderBy("started_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	executions, err := r.scanExecutions(query, args, ctx)
	if err != nil {
		return nil, 0, err
	}

	countQuery, countArgs := dbx.ST.
		Select("COUNT(id)").
		From("job_executions").
		Where(squirrel.Eq{"job_id": jobID}).
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	switch {
	case dbx.IsNoRows(err):
		total = 0
	case err != nil:
		return nil, 0, errors.Database(err)
	}

	return executions, total, nil
}

func (r *repository) GetPendingValidation(ctx context.Context) ([]*Execution, error) {
	query, args := dbx.ST.
		Select(
			"id", "job_id", "topic_id", "article_id",
			"prompt_id", "ai_provider_id", "ai_model", "category_id",
			"status", "error_message",
			"generation_time_ms", "tokens_used",
			"started_at", "generated_at", "validated_at", "published_at", "completed_at",
		).
		From("job_executions").
		Where(squirrel.Eq{"status": StatusPendingValidation}).
		OrderBy("generated_at ASC").
		MustSql()

	return r.scanExecutions(query, args, ctx)
}

func (r *repository) GetByStatus(ctx context.Context, status Status) ([]*Execution, error) {
	query, args := dbx.ST.
		Select(
			"id", "job_id", "topic_id", "article_id",
			"prompt_id", "ai_provider_id", "ai_model", "category_id",
			"status", "error_message",
			"generation_time_ms", "tokens_used",
			"started_at", "generated_at", "validated_at", "published_at", "completed_at",
		).
		From("job_executions").
		Where(squirrel.Eq{"status": status}).
		OrderBy("started_at DESC").
		MustSql()

	return r.scanExecutions(query, args, ctx)
}

func (r *repository) Update(ctx context.Context, exec *Execution) error {
	query, args := dbx.ST.
		Update("job_executions").
		Set("job_id", exec.JobID).
		Set("topic_id", exec.TopicID).
		Set("article_id", exec.ArticleID).
		Set("prompt_id", exec.PromptID).
		Set("ai_provider_id", exec.AIProviderID).
		Set("ai_model", exec.AIModel).
		Set("category_id", exec.CategoryID).
		Set("status", exec.Status).
		Set("error_message", exec.ErrorMessage).
		Set("generation_time_ms", exec.GenerationTimeMs).
		Set("tokens_used", exec.TokensUsed).
		Set("started_at", exec.StartedAt).
		Set("generated_at", exec.GeneratedAt).
		Set("validated_at", exec.ValidatedAt).
		Set("published_at", exec.PublishedAt).
		Set("completed_at", exec.CompletedAt).
		Where(squirrel.Eq{"id": exec.ID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsForeignKeyViolation(err):
		return errors.Validation("Invalid job, topic, article, prompt, AI provider or category ID")
	case err != nil:
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("job_execution", exec.ID)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("job_executions").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("job_execution", id)
	}

	return nil
}

func (r *repository) CountByJob(ctx context.Context, jobID int64) (int, error) {
	query, args := dbx.ST.
		Select("COUNT(id)").
		From("job_executions").
		Where(squirrel.Eq{"job_id": jobID}).
		MustSql()

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	switch {
	case dbx.IsNoRows(err):
		return 0, nil
	case err != nil:
		return 0, errors.Database(err)
	}

	return count, nil
}

func (r *repository) GetTotalCost(ctx context.Context, from, to time.Time) (float64, error) {
	// Note: This would need to be calculated based on tokens_used and AI provider costs
	// For now, returning 0 as cost calculation is complex
	return 0, nil
}

func (r *repository) GetTotalTokens(ctx context.Context, from, to time.Time) (int, error) {
	query, args := dbx.ST.
		Select("COALESCE(SUM(tokens_used), 0)").
		From("job_executions").
		Where(squirrel.GtOrEq{"started_at": from}).
		Where(squirrel.LtOrEq{"started_at": to}).
		MustSql()

	var total int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&total)
	switch {
	case dbx.IsNoRows(err):
		return 0, nil
	case err != nil:
		return 0, errors.Database(err)
	}

	return total, nil
}

func (r *repository) GetAverageGenerationTime(ctx context.Context, jobID int64) (int, error) {
	query, args := dbx.ST.
		Select("COALESCE(AVG(generation_time_ms), 0)").
		From("job_executions").
		Where(squirrel.Eq{"job_id": jobID}).
		Where(squirrel.NotEq{"generation_time_ms": nil}).
		MustSql()

	var avgTime int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&avgTime)
	switch {
	case dbx.IsNoRows(err):
		return 0, nil
	case err != nil:
		return 0, errors.Database(err)
	}

	return avgTime, nil
}

func (r *repository) scanExecution(query string, args []interface{}, ctx context.Context) (*Execution, error) {
	var exec Execution
	var articleID sql.NullInt64
	var errorMessage sql.NullString
	var generationTimeMs, tokensUsed sql.NullInt32
	var generatedAt, validatedAt, publishedAt, completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&exec.ID,
		&exec.JobID,
		&exec.TopicID,
		&articleID,
		&exec.PromptID,
		&exec.AIProviderID,
		&exec.AIModel,
		&exec.CategoryID,
		&exec.Status,
		&errorMessage,
		&generationTimeMs,
		&tokensUsed,
		&exec.StartedAt,
		&generatedAt,
		&validatedAt,
		&publishedAt,
		&completedAt,
	)

	if err != nil {
		return nil, err
	}

	if articleID.Valid {
		exec.ArticleID = &articleID.Int64
	}
	if errorMessage.Valid {
		exec.ErrorMessage = &errorMessage.String
	}
	if generationTimeMs.Valid {
		timeMs := int(generationTimeMs.Int32)
		exec.GenerationTimeMs = &timeMs
	}
	if tokensUsed.Valid {
		tokens := int(tokensUsed.Int32)
		exec.TokensUsed = &tokens
	}
	if generatedAt.Valid {
		exec.GeneratedAt = &generatedAt.Time
	}
	if validatedAt.Valid {
		exec.ValidatedAt = &validatedAt.Time
	}
	if publishedAt.Valid {
		exec.PublishedAt = &publishedAt.Time
	}
	if completedAt.Valid {
		exec.CompletedAt = &completedAt.Time
	}

	return &exec, nil
}

func (r *repository) scanExecutions(query string, args []interface{}, ctx context.Context) ([]*Execution, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var executions []*Execution
	for rows.Next() {
		var exec *Execution
		exec, err = r.scanExecutionFromRow(rows)
		if err != nil {
			return nil, errors.Database(err)
		}
		executions = append(executions, exec)
	}

	switch {
	case dbx.IsNoRows(err) || len(executions) == 0:
		return executions, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return executions, nil
}

func (r *repository) scanExecutionFromRow(rows *sql.Rows) (*Execution, error) {
	var exec Execution
	var articleID sql.NullInt64
	var errorMessage sql.NullString
	var generationTimeMs, tokensUsed sql.NullInt32
	var generatedAt, validatedAt, publishedAt, completedAt sql.NullTime

	err := rows.Scan(
		&exec.ID,
		&exec.JobID,
		&exec.TopicID,
		&articleID,
		&exec.PromptID,
		&exec.AIProviderID,
		&exec.AIModel,
		&exec.CategoryID,
		&exec.Status,
		&errorMessage,
		&generationTimeMs,
		&tokensUsed,
		&exec.StartedAt,
		&generatedAt,
		&validatedAt,
		&publishedAt,
		&completedAt,
	)

	if err != nil {
		return nil, err
	}

	if articleID.Valid {
		exec.ArticleID = &articleID.Int64
	}
	if errorMessage.Valid {
		exec.ErrorMessage = &errorMessage.String
	}
	if generationTimeMs.Valid {
		timeMs := int(generationTimeMs.Int32)
		exec.GenerationTimeMs = &timeMs
	}
	if tokensUsed.Valid {
		tokens := int(tokensUsed.Int32)
		exec.TokensUsed = &tokens
	}
	if generatedAt.Valid {
		exec.GeneratedAt = &generatedAt.Time
	}
	if validatedAt.Valid {
		exec.ValidatedAt = &validatedAt.Time
	}
	if publishedAt.Valid {
		exec.PublishedAt = &publishedAt.Time
	}
	if completedAt.Valid {
		exec.CompletedAt = &completedAt.Time
	}

	return &exec, nil
}
