package stats

import (
	"context"
	"database/sql"
	"fmt"
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
			WithScope("stats"),
	}
}

func (r *repository) IncrementSiteStats(ctx context.Context, siteID int64, date time.Time, field string, value int) error {
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	validFields := map[string]bool{
		"articles_published":     true,
		"articles_failed":        true,
		"total_words":            true,
		"internal_links_created": true,
		"external_links_created": true,
	}

	if !validFields[field] {
		return errors.Validation("Invalid statistics field: " + field)
	}

	query, args := dbx.ST.
		Insert("site_statistics").
		Columns("site_id", "date", field).
		Values(siteID, date, value).
		Suffix(fmt.Sprintf("ON CONFLICT(site_id, date) DO UPDATE SET %s = site_statistics.%s + EXCLUDED.%s", field, field, field)).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsForeignKeyViolation(err):
		return errors.Validation("Invalid site ID")
	case err != nil:
		return errors.Database(err)
	}

	return nil
}

func (r *repository) GetSiteStats(ctx context.Context, siteID int64, from, to time.Time) ([]*entities.SiteStats, error) {
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 999999999, to.Location())

	query, args := dbx.ST.
		Select(
			"id",
			"site_id",
			"date",
			"articles_published",
			"articles_failed",
			"total_words",
			"internal_links_created",
			"external_links_created",
		).
		From("site_statistics").
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

	var stats []*entities.SiteStats
	for rows.Next() {
		var stat entities.SiteStats
		err = rows.Scan(
			&stat.ID,
			&stat.SiteID,
			&stat.Date,
			&stat.ArticlesPublished,
			&stat.ArticlesFailed,
			&stat.TotalWords,
			&stat.InternalLinksCreated,
			&stat.ExternalLinksCreated,
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

func (r *repository) GetTotalSiteStats(ctx context.Context, siteID int64) (*entities.SiteStats, error) {
	query, args := dbx.ST.
		Select(
			"SUM(articles_published)",
			"SUM(articles_failed)",
			"SUM(total_words)",
			"SUM(internal_links_created)",
			"SUM(external_links_created)",
		).
		From("site_statistics").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	var stat entities.SiteStats
	var articlesPublished, articlesFailed, totalWords, internalLinks, externalLinks sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&articlesPublished,
		&articlesFailed,
		&totalWords,
		&internalLinks,
		&externalLinks,
	)

	switch {
	case dbx.IsNoRows(err):
		return &entities.SiteStats{
			SiteID:               siteID,
			ArticlesPublished:    0,
			ArticlesFailed:       0,
			TotalWords:           0,
			InternalLinksCreated: 0,
			ExternalLinksCreated: 0,
		}, nil
	case err != nil:
		return nil, errors.Database(err)
	}

	stat.SiteID = siteID
	stat.ArticlesPublished = int(articlesPublished.Int64)
	stat.ArticlesFailed = int(articlesFailed.Int64)
	stat.TotalWords = int(totalWords.Int64)
	stat.InternalLinksCreated = int(internalLinks.Int64)
	stat.ExternalLinksCreated = int(externalLinks.Int64)

	return &stat, nil
}

func (r *repository) GetGlobalStats(ctx context.Context, from, to time.Time) ([]*entities.SiteStats, error) {
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 999999999, to.Location())

	query, args := dbx.ST.
		Select(
			"date",
			"SUM(articles_published)",
			"SUM(articles_failed)",
			"SUM(total_words)",
			"SUM(internal_links_created)",
			"SUM(external_links_created)",
		).
		From("site_statistics").
		Where(squirrel.GtOrEq{"date": from}).
		Where(squirrel.LtOrEq{"date": to}).
		GroupBy("date").
		OrderBy("date ASC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var stats []*entities.SiteStats
	for rows.Next() {
		var stat entities.SiteStats
		var articlesPublished, articlesFailed, totalWords, internalLinks, externalLinks sql.NullInt64
		err = rows.Scan(
			&stat.Date,
			&articlesPublished,
			&articlesFailed,
			&totalWords,
			&internalLinks,
			&externalLinks,
		)
		if err != nil {
			return nil, errors.Database(err)
		}
		stat.ArticlesPublished = int(articlesPublished.Int64)
		stat.ArticlesFailed = int(articlesFailed.Int64)
		stat.TotalWords = int(totalWords.Int64)
		stat.InternalLinksCreated = int(internalLinks.Int64)
		stat.ExternalLinksCreated = int(externalLinks.Int64)
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
