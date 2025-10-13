package site

import (
	"Postulator/internal/domain/entities"
	"Postulator/internal/infra/database"
	"Postulator/pkg/dbx"
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"

	"github.com/Masterminds/squirrel"
)

var _ ICategoryRepository = (*CategoryRepository)(nil)

type CategoryRepository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewCategoryRepository(c di.Container) (*CategoryRepository, error) {
	var db *database.DB

	if err := c.Resolve(&db); err != nil {
		return nil, err
	}

	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	return &CategoryRepository{
		db:     db,
		logger: l,
	}, nil
}

func (c *CategoryRepository) Create(ctx context.Context, category *entities.Category) error {
	query, args := dbx.ST.
		Insert("site_categories").
		Columns("site_id", "wp_category_id", "name", "slug", "count").
		Values(category.SiteID, category.WPCategoryID, category.Name, category.Slug, category.Count).
		Suffix("ON CONFLICT DO NOTHING").
		MustSql()

	_, err := c.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		c.logger.Debugf("category %s already exists (%d, %d)", category.Name, category.SiteID, category.WPCategoryID)
		return nil
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}

func (c *CategoryRepository) GetBySiteID(ctx context.Context, siteID int64) ([]*entities.Category, error) {
	query, args := dbx.ST.
		Select("id", "site_id", "wp_category_id", "name", "slug", "count", "created_at").
		From("site_categories").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Internal(err)
	}

	var categories []*entities.Category
	for rows.Next() {
		var category entities.Category
		if err = rows.Scan(
			&category.ID,
			&category.SiteID,
			&category.WPCategoryID,
			&category.Name,
			&category.Slug,
			&category.Count,
			&category.CreatedAt,
		); err != nil {
			return nil, errors.Internal(err)
		}

		categories = append(categories, &category)
	}

	if err = rows.Err(); err != nil {
		switch {
		case dbx.IsNoRows(err):
			return nil, errors.NotFound("category", siteID)
		case err != nil:
			return nil, errors.Internal(err)
		}
	}

	return categories, nil
}

func (c *CategoryRepository) DeleteBySiteID(ctx context.Context, siteID int64) error {
	query, args := dbx.ST.
		Delete("site_categories").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	res, err := c.db.ExecContext(ctx, query, args...)
	switch {
	case res != nil:
		var affected int64
		if affected, err = res.RowsAffected(); err == nil && affected == 0 || dbx.IsNoRows(err) {
			return errors.NotFound("category", siteID)
		}
		return errors.Database(err)
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}
