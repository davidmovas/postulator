package sites

import (
	"Postulator/internal/infra/database"
	"Postulator/pkg/dbx"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
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
			WithScope("sites"),
	}
}

func (r *repository) Create(ctx context.Context, site *Site) error {
	query, args := dbx.ST.
		Insert("sites").
		Columns("name", "url", "wp_username", "wp_password", "status", "health_status").
		Values(site.Name, site.URL, site.WPUsername, site.WPPassword, site.Status, site.HealthStatus).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists("site")
	case err != nil:
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}

	site.ID = id
	return nil
}

func (r *repository) GetByID(ctx context.Context, id int64) (*Site, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"name",
			"url",
			"wp_username",
			"wp_password",
			"status",
			"last_health_check",
			"health_status",
			"created_at",
			"updated_at",
		).
		From("sites").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var site Site
	var lastHealthCheck sql.NullTime

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&site.ID,
		&site.Name,
		&site.URL,
		&site.WPUsername,
		&site.WPPassword,
		&site.Status,
		&lastHealthCheck,
		&site.HealthStatus,
		&site.CreatedAt,
		&site.UpdatedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("site", id)
	case err != nil:
		return nil, errors.Database(err)
	}

	if lastHealthCheck.Valid {
		site.LastHealthCheck = &lastHealthCheck.Time
	}

	return &site, nil
}

func (r *repository) GetAll(ctx context.Context) ([]*Site, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"name",
			"url",
			"wp_username",
			"wp_password",
			"status",
			"last_health_check",
			"health_status",
			"created_at",
			"updated_at",
		).
		From("sites").
		OrderBy("created_at DESC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var sites []*Site
	for rows.Next() {
		var site Site
		var lastHealthCheck sql.NullTime

		err = rows.Scan(
			&site.ID,
			&site.Name,
			&site.URL,
			&site.WPUsername,
			&site.WPPassword,
			&site.Status,
			&lastHealthCheck,
			&site.HealthStatus,
			&site.CreatedAt,
			&site.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Database(err)
		}

		if lastHealthCheck.Valid {
			site.LastHealthCheck = &lastHealthCheck.Time
		}

		sites = append(sites, &site)
	}

	switch {
	case dbx.IsNoRows(err) || len(sites) == 0:
		return sites, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return sites, nil
}

func (r *repository) GetByStatus(ctx context.Context, status Status) ([]*Site, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"name",
			"url",
			"wp_username",
			"wp_password",
			"status",
			"last_health_check",
			"health_status",
			"created_at",
			"updated_at",
		).
		From("sites").
		Where(squirrel.Eq{"status": status}).
		OrderBy("created_at DESC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var sites []*Site
	for rows.Next() {
		var site Site
		var lastHealthCheck sql.NullTime

		err = rows.Scan(
			&site.ID,
			&site.Name,
			&site.URL,
			&site.WPUsername,
			&site.WPPassword,
			&site.Status,
			&lastHealthCheck,
			&site.HealthStatus,
			&site.CreatedAt,
			&site.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Database(err)
		}

		if lastHealthCheck.Valid {
			site.LastHealthCheck = &lastHealthCheck.Time
		}

		sites = append(sites, &site)
	}

	switch {
	case dbx.IsNoRows(err) || len(sites) == 0:
		return sites, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return sites, nil
}

func (r *repository) Update(ctx context.Context, site *Site) error {
	query, args := dbx.ST.
		Update("sites").
		Set("name", site.Name).
		Set("url", site.URL).
		Set("wp_username", site.WPUsername).
		Set("wp_password", site.WPPassword).
		Set("status", site.Status).
		Set("health_status", site.HealthStatus).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": site.ID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists("site")
	case err != nil:
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("site", site.ID)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("sites").
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
		return errors.NotFound("site", id)
	}

	return nil
}

func (r *repository) UpdateHealthStatus(ctx context.Context, id int64, status HealthStatus, checkedAt time.Time) error {
	query, args := dbx.ST.
		Update("sites").
		Set("health_status", status).
		Set("last_health_check", checkedAt).
		Set("updated_at", time.Now()).
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
		return errors.NotFound("site", id)
	}

	return nil
}
