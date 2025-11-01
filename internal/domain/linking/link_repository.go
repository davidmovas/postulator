package linking

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/pkg/dbx"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ LinkRepository = (*linkRepository)(nil)

type linkRepository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewLinkRepository(db *database.DB, logger *logger.Logger) LinkRepository {
	return &linkRepository{
		db: db,
		logger: logger.
			WithScope("repository").
			WithScope("article_links"),
	}
}

func (r *linkRepository) CreateLink(ctx context.Context, link *Link) error {
	query, args := dbx.ST.
		Insert("article_links").
		Columns(
			"article_id", "link_type", "target_article_id",
			"url", "anchor_text", "position", "task_id",
		).
		Values(
			link.ArticleID, link.LinkType, link.TargetArticleID,
			link.URL, link.AnchorText, link.Position, link.TaskID,
		).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsForeignKeyViolation(err):
		return errors.Validation("Invalid article or task ID")
	case err != nil:
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}

	link.ID = id
	return nil
}

func (r *linkRepository) GetByArticleID(ctx context.Context, articleID int64) ([]*Link, error) {
	query, args := dbx.ST.
		Select(
			"id", "article_id", "link_type", "target_article_id",
			"url", "anchor_text", "position", "task_id", "created_at",
		).
		From("article_links").
		Where(squirrel.Eq{"article_id": articleID}).
		OrderBy("position ASC").
		MustSql()

	return r.scanLinks(query, args, ctx)
}

func (r *linkRepository) GetByTaskID(ctx context.Context, taskID int64) ([]*Link, error) {
	query, args := dbx.ST.
		Select(
			"id", "article_id", "link_type", "target_article_id",
			"url", "anchor_text", "position", "task_id", "created_at",
		).
		From("article_links").
		Where(squirrel.Eq{"task_id": taskID}).
		OrderBy("article_id ASC", "position ASC").
		MustSql()

	return r.scanLinks(query, args, ctx)
}

func (r *linkRepository) Update(ctx context.Context, link *Link) error {
	query, args := dbx.ST.
		Update("article_links").
		Set("article_id", link.ArticleID).
		Set("link_type", link.LinkType).
		Set("target_article_id", link.TargetArticleID).
		Set("url", link.URL).
		Set("anchor_text", link.AnchorText).
		Set("position", link.Position).
		Set("task_id", link.TaskID).
		Where(squirrel.Eq{"id": link.ID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsForeignKeyViolation(err):
		return errors.Validation("Invalid article or task ID")
	case err != nil:
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("article_link", link.ID)
	}

	return nil
}

func (r *linkRepository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("article_links").
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
		return errors.NotFound("article_link", id)
	}

	return nil
}

func (r *linkRepository) DeleteByArticleID(ctx context.Context, articleID int64) error {
	query, args := dbx.ST.
		Delete("article_links").
		Where(squirrel.Eq{"article_id": articleID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	return nil
}
