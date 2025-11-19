package healthcheck

import (
	"context"
	"database/sql"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/pkg/dbx"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"

	"github.com/Masterminds/squirrel"
)

type repository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewRepository(db *database.DB, logger *logger.Logger) Repository {
	return &repository{
		db: db,
		logger: logger.
			WithScope("repository").
			WithScope("healthcheck"),
	}
}

func (r *repository) SaveHistory(ctx context.Context, history *entities.HealthCheckHistory) error {
	query, args := dbx.ST.
		Insert("health_check_history").
		Columns("site_id", "checked_at", "status", "response_time_ms", "status_code", "error_message").
		Values(
			history.SiteID,
			history.CheckedAt,
			history.Status,
			history.ResponseTimeMs,
			history.StatusCode,
			history.ErrorMessage,
		).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}

	history.ID = id
	return nil
}

func (r *repository) GetHistoryBySite(ctx context.Context, siteID int64, limit int) ([]*entities.HealthCheckHistory, error) {
	query, args := dbx.ST.
		Select("id", "site_id", "checked_at", "status", "response_time_ms", "status_code", "error_message").
		From("health_check_history").
		Where(squirrel.Eq{"site_id": siteID}).
		OrderBy("checked_at DESC").
		Limit(uint64(limit)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var history []*entities.HealthCheckHistory
	for rows.Next() {
		var h entities.HealthCheckHistory
		var errorMsg sql.NullString

		err := rows.Scan(
			&h.ID,
			&h.SiteID,
			&h.CheckedAt,
			&h.Status,
			&h.ResponseTimeMs,
			&h.StatusCode,
			&errorMsg,
		)
		if err != nil {
			return nil, errors.Database(err)
		}

		if errorMsg.Valid {
			h.ErrorMessage = errorMsg.String
		}

		history = append(history, &h)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Database(err)
	}

	return history, nil
}

func (r *repository) GetHistoryBySitePeriod(ctx context.Context, siteID int64, from, to time.Time, limit, offset int) ([]*entities.HealthCheckHistory, int, error) {
    // total count first
    countQ, countArgs := dbx.ST.
        Select("COUNT(id)").
        From("health_check_history").
        Where(squirrel.Eq{"site_id": siteID}).
        Where(squirrel.And{
            squirrel.GtOrEq{"checked_at": from},
            squirrel.LtOrEq{"checked_at": to},
        }).
        MustSql()

    var total int
    if err := r.db.QueryRowContext(ctx, countQ, countArgs...).Scan(&total); err != nil {
        return nil, 0, errors.Database(err)
    }

    // items page
    if limit < 0 {
        limit = 0
    }
    if offset < 0 {
        offset = 0
    }

    sel := dbx.ST.
        Select("id", "site_id", "checked_at", "status", "response_time_ms", "status_code", "error_message").
        From("health_check_history").
        Where(squirrel.Eq{"site_id": siteID}).
        Where(squirrel.And{
            squirrel.GtOrEq{"checked_at": from},
            squirrel.LtOrEq{"checked_at": to},
        }).
        OrderBy("checked_at DESC")
    if limit > 0 {
        sel = sel.Limit(uint64(limit)).Offset(uint64(offset))
    }
    query, args := sel.MustSql()

    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, 0, errors.Database(err)
    }
    defer func() { _ = rows.Close() }()

    var history []*entities.HealthCheckHistory
    for rows.Next() {
        var h entities.HealthCheckHistory
        var errorMsg sql.NullString
        if err := rows.Scan(
            &h.ID,
            &h.SiteID,
            &h.CheckedAt,
            &h.Status,
            &h.ResponseTimeMs,
            &h.StatusCode,
            &errorMsg,
        ); err != nil {
            return nil, 0, errors.Database(err)
        }
        if errorMsg.Valid {
            h.ErrorMessage = errorMsg.String
        }
        history = append(history, &h)
    }
    if err = rows.Err(); err != nil {
        return nil, 0, errors.Database(err)
    }
    return history, total, nil
}

func (r *repository) GetLastCheckBySite(ctx context.Context, siteID int64) (*entities.HealthCheckHistory, error) {
	query, args := dbx.ST.
		Select("id", "site_id", "checked_at", "status", "response_time_ms", "status_code", "error_message").
		From("health_check_history").
		Where(squirrel.Eq{"site_id": siteID}).
		OrderBy("checked_at DESC").
		Limit(1).
		MustSql()

	var h entities.HealthCheckHistory
	var errorMsg sql.NullString

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&h.ID,
		&h.SiteID,
		&h.CheckedAt,
		&h.Status,
		&h.ResponseTimeMs,
		&h.StatusCode,
		&errorMsg,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, nil
	case err != nil:
		return nil, errors.Database(err)
	}

	if errorMsg.Valid {
		h.ErrorMessage = errorMsg.String
	}

	return &h, nil
}
