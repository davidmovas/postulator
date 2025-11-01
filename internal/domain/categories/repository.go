package categories

import (
	"context"
	"database/sql"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/pkg/dbx"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
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
			WithScope("categories"),
	}
}

func (r *repository) Create(ctx context.Context, category *Category) error {
	query, args := dbx.ST.
		Insert("categories").
		Columns(
			"site_id",
			"wp_category_id",
			"name",
			"slug",
			"description",
			"count",
		).
		Values(category.SiteID, category.WPCategoryID, category.Name, category.Slug, category.Description, category.Count).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists("category")
	case dbx.IsForeignKeyViolation(err):
		return errors.Validation("Invalid site ID")
	case err != nil:
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}

	category.ID = id
	return nil
}

func (r *repository) GetByID(ctx context.Context, id int64) (*Category, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"site_id",
			"wp_category_id",
			"name",
			"slug",
			"description",
			"count",
			"created_at",
			"updated_at",
		).
		From("categories").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var category Category
	var slug, description sql.NullString

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&category.ID,
		&category.SiteID,
		&category.WPCategoryID,
		&category.Name,
		&slug,
		&description,
		&category.Count,
		&category.CreatedAt,
		&category.UpdatedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("category", id)
	case err != nil:
		return nil, errors.Database(err)
	}

	if slug.Valid {
		category.Slug = &slug.String
	}
	if description.Valid {
		category.Description = &description.String
	}

	return &category, nil
}

func (r *repository) GetBySiteID(ctx context.Context, siteID int64) ([]*Category, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"site_id",
			"wp_category_id",
			"name",
			"slug",
			"description",
			"count",
			"created_at",
			"updated_at",
		).
		From("categories").
		Where(squirrel.Eq{"site_id": siteID}).
		OrderBy("name ASC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var categories []*Category
	for rows.Next() {
		var category Category
		var slug, description sql.NullString

		err = rows.Scan(
			&category.ID,
			&category.SiteID,
			&category.WPCategoryID,
			&category.Name,
			&slug,
			&description,
			&category.Count,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Database(err)
		}

		if slug.Valid {
			category.Slug = &slug.String
		}
		if description.Valid {
			category.Description = &description.String
		}

		categories = append(categories, &category)
	}

	switch {
	case dbx.IsNoRows(err) || len(categories) == 0:
		return categories, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return categories, nil
}

func (r *repository) GetByWPCategoryID(ctx context.Context, siteID int64, wpCategoryID int) (*Category, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"site_id",
			"wp_category_id",
			"name",
			"slug",
			"description",
			"count",
			"created_at",
			"updated_at",
		).
		From("categories").
		Where(squirrel.Eq{"site_id": siteID, "wp_category_id": wpCategoryID}).
		MustSql()

	var category Category
	var slug, description sql.NullString

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&category.ID,
		&category.SiteID,
		&category.WPCategoryID,
		&category.Name,
		&slug,
		&description,
		&category.Count,
		&category.CreatedAt,
		&category.UpdatedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("category", wpCategoryID)
	case err != nil:
		return nil, errors.Database(err)
	}

	if slug.Valid {
		category.Slug = &slug.String
	}
	if description.Valid {
		category.Description = &description.String
	}

	return &category, nil
}

func (r *repository) Update(ctx context.Context, category *Category) error {
	query, args := dbx.ST.
		Update("categories").
		Set("name", category.Name).
		Set("slug", category.Slug).
		Set("description", category.Description).
		Set("count", category.Count).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": category.ID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists("category")
	case err != nil:
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("category", category.ID)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("categories").
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
		return errors.NotFound("category", id)
	}

	return nil
}

func (r *repository) DeleteBySiteID(ctx context.Context, siteID int64) error {
	query, args := dbx.ST.
		Delete("categories").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	return nil
}

func (r *repository) BulkUpsert(ctx context.Context, siteID int64, categories []*Category) error {
	if len(categories) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Database(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, category := range categories {
		query, args := dbx.ST.
			Insert("categories").
			Columns("site_id", "wp_category_id", "name", "slug", "description", "count").
			Values(siteID, category.WPCategoryID, category.Name, category.Slug, category.Description, category.Count).
			Suffix("ON CONFLICT(site_id, wp_category_id) DO UPDATE SET name = EXCLUDED.name, slug = EXCLUDED.slug, description = EXCLUDED.description, count = EXCLUDED.count, updated_at = CURRENT_TIMESTAMP").
			MustSql()

		_, err = tx.ExecContext(ctx, query, args...)
		if err != nil {
			return errors.Database(err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Database(err)
	}

	return nil
}
