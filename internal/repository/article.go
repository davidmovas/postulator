package repository

import (
	"Postulator/internal/models"
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
)

func (r *Repository) GetArticles(ctx context.Context, limit int, offset int) (*models.PaginationResult[*models.Article], error) {
	// Get articles
	query, args := builder.
		Select(
			"id",
			"site_id",
			"topic_id",
			"title",
			"content",
			"excerpt",
			"keywords",
			"tags",
			"category",
			"status",
			"wordpress_id",
			"gpt_model",
			"tokens",
			"slug",
			"outline",
			"error_msg",
			"created_at",
			"published_at",
		).
		From("articles").
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var articles []*models.Article
	for rows.Next() {
		var article models.Article
		if err = rows.Scan(
			&article.ID,
			&article.SiteID,
			&article.TopicID,
			&article.Title,
			&article.Content,
			&article.Excerpt,
			&article.Keywords,
			&article.Tags,
			&article.Category,
			&article.Status,
			&article.WordPressID,
			&article.GPTModel,
			&article.Tokens,
			&article.Slug,
			&article.Outline,
			&article.ErrorMsg,
			&article.CreatedAt,
			&article.PublishedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		articles = append(articles, &article)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(*)").
		From("articles").
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count articles: %w", err)
	}

	return &models.PaginationResult[*models.Article]{
		Data:   articles,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (r *Repository) GetArticle(ctx context.Context, id int64) (*models.Article, error) {
	query, args := builder.
		Select(
			"id",
			"site_id",
			"topic_id",
			"title",
			"content",
			"excerpt",
			"keywords",
			"tags",
			"category",
			"status",
			"wordpress_id",
			"gpt_model",
			"tokens",
			"slug",
			"outline",
			"error_msg",
			"created_at",
			"published_at",
		).
		From("articles").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var article models.Article
	if err := r.db.QueryRowContext(ctx, query, args...).
		Scan(
			&article.ID,
			&article.SiteID,
			&article.TopicID,
			&article.Title,
			&article.Content,
			&article.Excerpt,
			&article.Keywords,
			&article.Tags,
			&article.Category,
			&article.Status,
			&article.WordPressID,
			&article.GPTModel,
			&article.Tokens,
			&article.Slug,
			&article.Outline,
			&article.ErrorMsg,
			&article.CreatedAt,
			&article.PublishedAt,
		); err != nil {
		return nil, fmt.Errorf("failed to query article: %w", err)
	}

	return &article, nil
}

func (r *Repository) GetArticlesBySiteID(ctx context.Context, siteID int64, limit int, offset int) (*models.PaginationResult[*models.Article], error) {
	query, args := builder.
		Select(
			"id",
			"site_id",
			"topic_id",
			"title",
			"content",
			"excerpt",
			"keywords",
			"tags",
			"category",
			"status",
			"wordpress_id",
			"gpt_model",
			"tokens",
			"slug",
			"outline",
			"error_msg",
			"created_at",
			"published_at",
		).
		From("articles").
		Where(squirrel.Eq{"site_id": siteID}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles by site: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var articles []*models.Article
	for rows.Next() {
		var article models.Article
		if err = rows.Scan(
			&article.ID,
			&article.SiteID,
			&article.TopicID,
			&article.Title,
			&article.Content,
			&article.Excerpt,
			&article.Keywords,
			&article.Tags,
			&article.Category,
			&article.Status,
			&article.WordPressID,
			&article.GPTModel,
			&article.Tokens,
			&article.Slug,
			&article.Outline,
			&article.ErrorMsg,
			&article.CreatedAt,
			&article.PublishedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		articles = append(articles, &article)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(*)").
		From("articles").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count articles by site: %w", err)
	}

	return &models.PaginationResult[*models.Article]{
		Data:   articles,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (r *Repository) CreateArticle(ctx context.Context, article *models.Article) (*models.Article, error) {
	query, args := builder.
		Insert("articles").
		Columns(
			"site_id",
			"topic_id",
			"title",
			"content",
			"excerpt",
			"keywords",
			"tags",
			"category",
			"status",
			"wordpress_id",
			"gpt_model",
			"tokens",
			"slug",
			"outline",
			"error_msg",
			"created_at",
			"published_at",
		).
		Values(
			article.SiteID,
			article.TopicID,
			article.Title,
			article.Content,
			article.Excerpt,
			article.Keywords,
			article.Tags,
			article.Category,
			article.Status,
			article.WordPressID,
			article.GPTModel,
			article.Tokens,
			article.Slug,
			article.Outline,
			article.ErrorMsg,
			article.CreatedAt,
			article.PublishedAt,
		).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create article: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get insert id: %w", err)
	}

	article.ID = id
	return article, nil
}

func (r *Repository) UpdateArticle(ctx context.Context, article *models.Article) (*models.Article, error) {

	query, args := builder.
		Update("articles").
		Set("site_id", article.SiteID).
		Set("topic_id", article.TopicID).
		Set("title", article.Title).
		Set("content", article.Content).
		Set("excerpt", article.Excerpt).
		Set("keywords", article.Keywords).
		Set("tags", article.Tags).
		Set("category", article.Category).
		Set("status", article.Status).
		Set("wordpress_id", article.WordPressID).
		Set("gpt_model", article.GPTModel).
		Set("tokens", article.Tokens).
		Set("slug", article.Slug).
		Set("outline", article.Outline).
		Set("error_msg", article.ErrorMsg).
		Set("published_at", article.PublishedAt).
		Where(squirrel.Eq{"id": article.ID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update article: %w", err)
	}

	return article, nil
}

func (r *Repository) DeleteArticle(ctx context.Context, id int64) error {
	query, args := builder.
		Delete("articles").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete article: %w", err)
	}

	return nil
}

func (r *Repository) GetArticleByHash(ctx context.Context, hash string) (*models.Article, error) {
	// For now, we'll implement this as a simple search by a combination of fields
	// In a real implementation, you might want to add a hash column to the articles table
	query, args := builder.
		Select(
			"id",
			"site_id",
			"topic_id",
			"title",
			"content",
			"excerpt",
			"keywords",
			"tags",
			"category",
			"status",
			"wordpress_id",
			"gpt_model",
			"tokens",
			"slug",
			"outline",
			"error_msg",
			"created_at",
			"published_at",
		).
		From("articles").
		Where(squirrel.Like{"title": "%" + hash[:10] + "%"}). // Simple hash matching for now
		Limit(1).
		MustSql()

	var article models.Article
	if err := r.db.QueryRowContext(ctx, query, args...).
		Scan(
			&article.ID,
			&article.SiteID,
			&article.TopicID,
			&article.Title,
			&article.Content,
			&article.Excerpt,
			&article.Keywords,
			&article.Tags,
			&article.Category,
			&article.Status,
			&article.WordPressID,
			&article.GPTModel,
			&article.Tokens,
			&article.Slug,
			&article.Outline,
			&article.ErrorMsg,
			&article.CreatedAt,
			&article.PublishedAt,
		); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil // No article found, not an error
		}
		return nil, fmt.Errorf("failed to query article by hash: %w", err)
	}

	return &article, nil
}

func (r *Repository) GetArticleBySlug(ctx context.Context, siteID int64, slug string) (*models.Article, error) {
	query, args := builder.
		Select(
			"id",
			"site_id",
			"topic_id",
			"title",
			"content",
			"excerpt",
			"keywords",
			"tags",
			"category",
			"status",
			"wordpress_id",
			"gpt_model",
			"tokens",
			"slug",
			"outline",
			"error_msg",
			"created_at",
			"published_at",
		).
		From("articles").
		Where(squirrel.And{
			squirrel.Eq{"site_id": siteID},
			squirrel.Eq{"slug": slug},
		}).
		MustSql()

	var article models.Article
	if err := r.db.QueryRowContext(ctx, query, args...).
		Scan(
			&article.ID,
			&article.SiteID,
			&article.TopicID,
			&article.Title,
			&article.Content,
			&article.Excerpt,
			&article.Keywords,
			&article.Tags,
			&article.Category,
			&article.Status,
			&article.WordPressID,
			&article.GPTModel,
			&article.Tokens,
			&article.Slug,
			&article.Outline,
			&article.ErrorMsg,
			&article.CreatedAt,
			&article.PublishedAt,
		); err != nil {
		return nil, fmt.Errorf("failed to query article by slug: %w", err)
	}

	return &article, nil
}
