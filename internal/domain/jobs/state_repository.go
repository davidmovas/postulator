package jobs

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/pkg/dbx"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ StateRepository = (*stateRepository)(nil)

type stateRepository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewStateRepository(db *database.DB, logger *logger.Logger) StateRepository {
	return &stateRepository{
		db: db,
		logger: logger.
			WithScope("repository").
			WithScope("job_state"),
	}
}

func (r *stateRepository) Get(ctx context.Context, jobID int64) (*entities.State, error) {
	query, args := dbx.ST.
		Select(
			"job_id", "last_run_at", "next_run_at",
			"total_executions", "failed_executions", "last_category_index",
		).
		From("job_state").
		Where(squirrel.Eq{"job_id": jobID}).
		MustSql()

	var state entities.State
	var lastRunAt, nextRunAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&state.JobID,
		&lastRunAt,
		&nextRunAt,
		&state.TotalExecutions,
		&state.FailedExecutions,
		&state.LastCategoryIndex,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("job_state", jobID)
	case err != nil:
		return nil, errors.Database(err)
	}

	if lastRunAt.Valid {
		state.LastRunAt = &lastRunAt.Time
	}
	if nextRunAt.Valid {
		state.NextRunAt = &nextRunAt.Time
	}

	return &state, nil
}

func (r *stateRepository) Update(ctx context.Context, state *entities.State) error {
	query, args := dbx.ST.
		Insert("job_state").
		Columns(
			"job_id", "last_run_at", "next_run_at",
			"total_executions", "failed_executions", "last_category_index",
		).
		Values(
			state.JobID, state.LastRunAt, state.NextRunAt,
			state.TotalExecutions, state.FailedExecutions, state.LastCategoryIndex,
		).
		Suffix("ON CONFLICT(job_id) DO UPDATE SET last_run_at = EXCLUDED.last_run_at, next_run_at = EXCLUDED.next_run_at, total_executions = EXCLUDED.total_executions, failed_executions = EXCLUDED.failed_executions, last_category_index = EXCLUDED.last_category_index").
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	return nil
}

func (r *stateRepository) UpdateNextRun(ctx context.Context, jobID int64, nextRun *time.Time) error {
	query, args := dbx.ST.
		Update("job_state").
		Set("next_run_at", nextRun).
		Where(squirrel.Eq{"job_id": jobID}).
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
		return errors.NotFound("job_state", jobID)
	}

	return nil
}

func (r *stateRepository) IncrementExecutions(ctx context.Context, jobID int64, failed bool) error {
	field := "total_executions"
	if failed {
		field = "failed_executions"
	}

	query, args := dbx.ST.
		Update("job_state").
		Set(field, squirrel.Expr(field+" + 1")).
		Where(squirrel.Eq{"job_id": jobID}).
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
		return errors.NotFound("job_state", jobID)
	}

	return nil
}

func (r *stateRepository) UpdateCategoryIndex(ctx context.Context, jobID int64, index int) error {
	query, args := dbx.ST.
		Update("job_state").
		Set("last_category_index", index).
		Where(squirrel.Eq{"job_id": jobID}).
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
		return errors.NotFound("job_state", jobID)
	}

	return nil
}
