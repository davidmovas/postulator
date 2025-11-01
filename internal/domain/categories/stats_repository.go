package categories

import (
	"Postulator/internal/infra/database"
	"Postulator/pkg/dbx"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"
	"time"

	"github.com/Masterminds/squirrel"
)

var _ StatisticsRepository = (*statsRepository)(nil)

type statsRepository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewStatsRepository(db *database.DB, logger *logger.Logger) StatisticsRepository {
	return &statsRepository{
		db: db,
		logger: logger.
			WithScope("repository").
			WithScope("category_stats"),
	}
}

func (r *statsRepository) Increment(ctx context.Context, siteID, categoryID int64, date time.Time, articlesPublished, totalWords int) error {
	query, args := dbx.ST.
		Insert("category_statistics").
		Columns("site_id", "category_id", "date", "articles_published", "total_words").
		Values(siteID, categoryID, date, articlesPublished, totalWords).
		Suffix("ON CONFLICT(site_id, category_id, date) DO UPDATE SET articles_published = category_statistics.articles_published + EXCLUDED.articles_published, total_words = category_statistics.total_words + EXCLUDED.total_words").
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsForeignKeyViolation(err):
		return errors.Validation("Invalid site or category ID")
	case err != nil:
		return errors.Database(err)
	}

	return nil
}

func (r *statsRepository) GetByCategory(ctx context.Context, categoryID int64, from, to time.Time) ([]*Statistics, error) {
	query, args := dbx.ST.
		Select("category_id", "date", "articles_published", "total_words").
		From("category_statistics").
		Where(squirrel.Eq{"category_id": categoryID}).
		Where(squirrel.GtOrEq{"date": from}).
		Where(squirrel.LtOrEq{"date": to}).
		OrderBy("date ASC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var stats []*Statistics
	for rows.Next() {
		var stat Statistics
		err = rows.Scan(
			&stat.CategoryID,
			&stat.Date,
			&stat.ArticlesPublished,
			&stat.TotalWords,
		)
		if err != nil {
			return nil, errors.Database(err)
		}
		stats = append(stats, &stat)
	}

	switch {
	case dbx.IsNoRows(err) || len(stats) == 0:
		return stats, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return stats, nil
}

func (r *statsRepository) GetBySite(ctx context.Context, siteID int64, from, to time.Time) ([]*Statistics, error) {
	query, args := dbx.ST.
		Select("category_id", "date", "articles_published", "total_words").
		From("category_statistics").
		Where(squirrel.Eq{"site_id": siteID}).
		Where(squirrel.GtOrEq{"date": from}).
		Where(squirrel.LtOrEq{"date": to}).
		OrderBy("date ASC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var stats []*Statistics
	for rows.Next() {
		var stat Statistics
		err = rows.Scan(
			&stat.CategoryID,
			&stat.Date,
			&stat.ArticlesPublished,
			&stat.TotalWords,
		)
		if err != nil {
			return nil, errors.Database(err)
		}
		stats = append(stats, &stat)
	}

	switch {
	case dbx.IsNoRows(err) || len(stats) == 0:
		return stats, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return stats, nil
}
