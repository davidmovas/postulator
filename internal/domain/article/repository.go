package article

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

type IRepository interface {
	Create(ctx context.Context, article *entities.Article) error
	GetByID(ctx context.Context, id int64) (*entities.Article, error)
	GetBySiteID(ctx context.Context, siteID int64) ([]*entities.Article, error)
	GetByJobID(ctx context.Context, jobID int64) ([]*entities.Article, error)
	Update(ctx context.Context, article *entities.Article) error
	Delete(ctx context.Context, id int64) error
}

var _ IRepository = (*Repository)(nil)

type Repository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewRepository(c di.Container) (*Repository, error) {
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

func (r *Repository) Create(ctx context.Context, article *entities.Article) error {
	query, args := dbx.ST.
		Insert("articles").
		Columns(
			"site_id",
			"job_id",
			"topic_id",
			"title",
			"original_title",
			"content",
			"excerpt",
			"wp_post_id",
			"wp_post_url",
			"wp_category_id",
			"status",
			"word_count",
			"published_at",
		).
		Values(
			article.SiteID,
			article.JobID,
			article.TopicID,
			article.Title,
			article.OriginalTitle,
			article.Content,
			article.Excerpt,
			article.WPPostID,
			article.WPPostURL,
			article.WPCategoryID,
			article.Status,
			article.WordCount,
			article.PublishedAt,
		).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Internal(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Internal(err)
	}

	article.ID = id
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*entities.Article, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"site_id",
			"job_id",
			"topic_id",
			"title",
			"original_title",
			"content",
			"excerpt",
			"wp_post_id",
			"wp_post_url",
			"wp_category_id",
			"status",
			"word_count",
			"created_at",
			"published_at",
		).
		From("articles").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var article entities.Article
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&article.ID,
		&article.SiteID,
		&article.JobID,
		&article.TopicID,
		&article.Title,
		&article.OriginalTitle,
		&article.Content,
		&article.Excerpt,
		&article.WPPostID,
		&article.WPPostURL,
		&article.WPCategoryID,
		&article.Status,
		&article.WordCount,
		&article.CreatedAt,
		&article.PublishedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("article", id)
	case err != nil:
		return nil, errors.Internal(err)
	}

	return &article, nil
}

func (r *Repository) GetBySiteID(ctx context.Context, siteID int64) ([]*entities.Article, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"site_id",
			"job_id",
			"topic_id",
			"title",
			"original_title",
			"content",
			"excerpt",
			"wp_post_id",
			"wp_post_url",
			"wp_category_id",
			"status",
			"word_count",
			"created_at",
			"published_at",
		).
		From("articles").
		Where(squirrel.Eq{"site_id": siteID}).
		OrderBy("published_at DESC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Internal(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var articles []*entities.Article
	for rows.Next() {
		var article entities.Article
		if err = rows.Scan(
			&article.ID,
			&article.SiteID,
			&article.JobID,
			&article.TopicID,
			&article.Title,
			&article.OriginalTitle,
			&article.Content,
			&article.Excerpt,
			&article.WPPostID,
			&article.WPPostURL,
			&article.WPCategoryID,
			&article.Status,
			&article.WordCount,
			&article.CreatedAt,
			&article.PublishedAt,
		); err != nil {
			return nil, errors.Internal(err)
		}

		articles = append(articles, &article)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Internal(err)
	}

	return articles, nil
}

func (r *Repository) GetByJobID(ctx context.Context, jobID int64) ([]*entities.Article, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"site_id",
			"job_id",
			"topic_id",
			"title",
			"original_title",
			"content",
			"excerpt",
			"wp_post_id",
			"wp_post_url",
			"wp_category_id",
			"status",
			"word_count",
			"created_at",
			"published_at",
		).
		From("articles").
		Where(squirrel.Eq{"job_id": jobID}).
		OrderBy("published_at DESC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Internal(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var articles []*entities.Article
	for rows.Next() {
		var article entities.Article
		if err = rows.Scan(
			&article.ID,
			&article.SiteID,
			&article.JobID,
			&article.TopicID,
			&article.Title,
			&article.OriginalTitle,
			&article.Content,
			&article.Excerpt,
			&article.WPPostID,
			&article.WPPostURL,
			&article.WPCategoryID,
			&article.Status,
			&article.WordCount,
			&article.CreatedAt,
			&article.PublishedAt,
		); err != nil {
			return nil, errors.Internal(err)
		}

		articles = append(articles, &article)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Internal(err)
	}

	return articles, nil
}

func (r *Repository) Update(ctx context.Context, article *entities.Article) error {
	query, args := dbx.ST.
		Update("articles").
		Set("title", article.Title).
		Set("content", article.Content).
		Set("excerpt", article.Excerpt).
		Set("status", article.Status).
		Set("word_count", article.WordCount).
		Set("published_at", article.PublishedAt).
		Where(squirrel.Eq{"id": article.ID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsNoRows(err):
		return errors.NotFound("article", article.ID)
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("articles").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Internal(err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return errors.Internal(err)
	}

	if affected == 0 {
		return errors.NotFound("article", id)
	}

	return nil
}
