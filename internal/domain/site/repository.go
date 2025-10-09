package site

import (
	"Postulator/internal/domain/entities"
	"Postulator/internal/infra/database"
	"Postulator/pkg/dbx"
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"
	"time"

	"github.com/Masterminds/squirrel"
)

var (
	_ ISiteRepository = (*Repository)(nil)
)

type Repository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewSiteRepository(c di.Container) (*Repository, error) {
	var db *database.DB

	if err := c.Resolve(&db); err != nil {
		return nil, err
	}

	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	return &Repository{
		db:     db,
		logger: l,
	}, nil
}

func (r *Repository) Create(ctx context.Context, site *entities.Site) error {
	query, args := dbx.ST.
		Insert("sites").
		Columns("name", "url", "wp_username", "wp_password", "status", "health_status").
		Values(site.Name, site.URL, site.WPUsername, site.WPPassword, site.Status, site.HealthStatus).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists(site.URL)
	case err != nil:
		return errors.Internal(err)
	}

	return err
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*entities.Site, error) {
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

	var site entities.Site
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&site.ID,
		&site.Name,
		&site.URL,
		&site.WPUsername,
		&site.WPPassword,
		&site.Status,
		&site.LastHealthCheck,
		&site.HealthStatus,
		&site.CreatedAt,
		&site.UpdatedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("site", id)
	case err != nil:
		return nil, errors.Internal(err)
	}

	return &site, nil
}

func (r *Repository) GetAll(ctx context.Context) ([]*entities.Site, error) {
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
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Internal(err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var sites []*entities.Site
	for rows.Next() {
		var site entities.Site
		if err = rows.Scan(
			&site.ID,
			&site.Name,
			&site.URL,
			&site.WPUsername,
			&site.WPPassword,
			&site.Status,
			&site.LastHealthCheck,
			&site.HealthStatus,
			&site.CreatedAt,
			&site.UpdatedAt,
		); err != nil {
			return nil, errors.Internal(err)
		}

		sites = append(sites, &site)
	}

	return sites, nil
}

func (r *Repository) Update(ctx context.Context, site *entities.Site) error {
	query, args := dbx.ST.
		Update("sites").
		Set("name", site.Name).
		Set("url", site.URL).
		Set("wp_username", site.WPUsername).
		Set("wp_password", site.WPPassword).
		Set("status", site.Status).
		Set("last_health_check", site.LastHealthCheck).
		Set("health_status", site.HealthStatus).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": site.ID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsNoRows(err):
		return errors.NotFound("site", site.ID)
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists(site.URL)
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("sites").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsNoRows(err):
		return errors.NotFound("site", id)
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}

func (r *Repository) UpdateHealthStatus(ctx context.Context, id int64, status entities.HealthStatus) error {
	query, args := dbx.ST.
		Update("sites").
		Set("health_status", status).
		Set("last_health_check", time.Now()).
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsNoRows(err):
		return errors.NotFound("site", id)
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}
