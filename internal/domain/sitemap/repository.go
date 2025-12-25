package sitemap

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
			WithScope("sitemap"),
	}
}

func (r *repository) Create(ctx context.Context, sitemap *entities.Sitemap) error {
	query, args := dbx.ST.
		Insert("sitemaps").
		Columns("site_id", "name", "description", "source", "status").
		Values(sitemap.SiteID, sitemap.Name, sitemap.Description, sitemap.Source, sitemap.Status).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}

	sitemap.ID = id
	return nil
}

func (r *repository) GetByID(ctx context.Context, id int64) (*entities.Sitemap, error) {
	query, args := dbx.ST.
		Select(
			"id", "site_id", "name", "description",
			"source", "status", "created_at", "updated_at",
		).
		From("sitemaps").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var sitemap entities.Sitemap
	var description sql.NullString

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&sitemap.ID,
		&sitemap.SiteID,
		&sitemap.Name,
		&description,
		&sitemap.Source,
		&sitemap.Status,
		&sitemap.CreatedAt,
		&sitemap.UpdatedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("sitemap", id)
	case err != nil:
		return nil, errors.Database(err)
	}

	if description.Valid {
		sitemap.Description = &description.String
	}

	return &sitemap, nil
}

func (r *repository) GetBySiteID(ctx context.Context, siteID int64) ([]*entities.Sitemap, error) {
	query, args := dbx.ST.
		Select(
			"id", "site_id", "name", "description",
			"source", "status", "created_at", "updated_at",
		).
		From("sitemaps").
		Where(squirrel.Eq{"site_id": siteID}).
		OrderBy("created_at DESC").
		MustSql()

	return r.querySitemaps(ctx, query, args)
}

func (r *repository) GetAll(ctx context.Context) ([]*entities.Sitemap, error) {
	query, args := dbx.ST.
		Select(
			"id", "site_id", "name", "description",
			"source", "status", "created_at", "updated_at",
		).
		From("sitemaps").
		OrderBy("created_at DESC").
		MustSql()

	return r.querySitemaps(ctx, query, args)
}

func (r *repository) querySitemaps(ctx context.Context, query string, args []any) ([]*entities.Sitemap, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var sitemaps []*entities.Sitemap
	for rows.Next() {
		var sitemap entities.Sitemap
		var description sql.NullString

		err = rows.Scan(
			&sitemap.ID,
			&sitemap.SiteID,
			&sitemap.Name,
			&description,
			&sitemap.Source,
			&sitemap.Status,
			&sitemap.CreatedAt,
			&sitemap.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Database(err)
		}

		if description.Valid {
			sitemap.Description = &description.String
		}

		sitemaps = append(sitemaps, &sitemap)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Database(err)
	}

	return sitemaps, nil
}

func (r *repository) Update(ctx context.Context, sitemap *entities.Sitemap) error {
	query, args := dbx.ST.
		Update("sitemaps").
		Set("name", sitemap.Name).
		Set("description", sitemap.Description).
		Set("source", sitemap.Source).
		Set("status", sitemap.Status).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": sitemap.ID}).
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
		return errors.NotFound("sitemap", sitemap.ID)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("sitemaps").
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
		return errors.NotFound("sitemap", id)
	}

	return nil
}

func (r *repository) UpdateStatus(ctx context.Context, id int64, status entities.SitemapStatus) error {
	query, args := dbx.ST.
		Update("sitemaps").
		Set("status", status).
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
		return errors.NotFound("sitemap", id)
	}

	return nil
}

func (r *repository) TouchUpdatedAt(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Update("sitemaps").
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	return nil
}
