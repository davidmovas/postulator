package articles

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/pkg/dbx"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"

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
			WithScope("articles"),
	}
}

func (r *repository) Create(ctx context.Context, article *entities.Article) error {
	categoryIDsJSON, err := json.Marshal(article.WPCategoryIDs)
	if err != nil {
		return errors.Validation("Invalid category IDs format")
	}

	query, args := dbx.ST.
		Insert("articles").
		Columns(
			"site_id", "job_id", "topic_id",
			"title", "original_title", "content", "excerpt",
			"wp_post_id", "wp_post_url", "wp_category_ids",
			"status", "source", "is_edited", "word_count",
			"published_at", "last_synced_at",
		).
		Values(
			article.SiteID, article.JobID, article.TopicID,
			article.Title, article.OriginalTitle, article.Content, article.Excerpt,
			article.WPPostID, article.WPPostURL, categoryIDsJSON,
			article.Status, article.Source, article.IsEdited, article.WordCount,
			article.PublishedAt, article.LastSyncedAt,
		).
		Suffix(`ON CONFLICT(site_id, wp_post_id) DO UPDATE SET
        title = EXCLUDED.title,
        content = EXCLUDED.content,
        excerpt = EXCLUDED.excerpt,
        wp_post_url = EXCLUDED.wp_post_url,
        wp_category_ids = EXCLUDED.wp_category_ids,
        status = EXCLUDED.status,
        is_edited = EXCLUDED.is_edited,
        word_count = EXCLUDED.word_count,
        published_at = EXCLUDED.published_at,
        last_synced_at = EXCLUDED.last_synced_at,
        updated_at = CURRENT_TIMESTAMP
		`).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}

	article.ID = id
	return nil
}

func (r *repository) GetByID(ctx context.Context, id int64) (*entities.Article, error) {
	query, args := dbx.ST.
		Select(
			"id", "site_id", "job_id", "topic_id",
			"title", "original_title", "content", "excerpt",
			"wp_post_id", "wp_post_url", "wp_category_ids",
			"status", "source", "is_edited", "word_count",
			"created_at", "published_at", "updated_at", "last_synced_at",
		).
		From("articles").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	article, err := r.scanArticle(query, args, ctx)
	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("article", id)
	case err != nil:
		return nil, errors.Database(err)
	}

	return article, nil
}

func (r *repository) GetByWPPostID(ctx context.Context, siteID int64, wpPostID int) (*entities.Article, error) {
	query, args := dbx.ST.
		Select(
			"id", "site_id", "job_id", "topic_id",
			"title", "original_title", "content", "excerpt",
			"wp_post_id", "wp_post_url", "wp_category_ids",
			"status", "source", "is_edited", "word_count",
			"created_at", "published_at", "updated_at", "last_synced_at",
		).
		From("articles").
		Where(squirrel.Eq{"site_id": siteID, "wp_post_id": wpPostID}).
		MustSql()

	article, err := r.scanArticle(query, args, ctx)
	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("article", wpPostID)
	case err != nil:
		return nil, errors.Database(err)
	}

	return article, nil
}

func (r *repository) Update(ctx context.Context, article *entities.Article) error {
	categoryIDsJSON, err := json.Marshal(article.WPCategoryIDs)
	if err != nil {
		return errors.Validation("Invalid category IDs format")
	}

	query, args := dbx.ST.
		Update("articles").
		Set("site_id", article.SiteID).
		Set("job_id", article.JobID).
		Set("topic_id", article.TopicID).
		Set("title", article.Title).
		Set("original_title", article.OriginalTitle).
		Set("content", article.Content).
		Set("excerpt", article.Excerpt).
		Set("wp_post_id", article.WPPostID).
		Set("wp_post_url", article.WPPostURL).
		Set("wp_category_ids", categoryIDsJSON).
		Set("status", article.Status).
		Set("source", article.Source).
		Set("is_edited", article.IsEdited).
		Set("word_count", article.WordCount).
		Set("published_at", article.PublishedAt).
		Set("last_synced_at", article.LastSyncedAt).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": article.ID}).
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
		return errors.NotFound("article", article.ID)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("articles").
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
		return errors.NotFound("article", id)
	}

	return nil
}

func (r *repository) ListBySite(ctx context.Context, siteID int64, limit, offset int) ([]*entities.Article, error) {
	query, args := dbx.ST.
		Select(
			"id", "site_id", "job_id", "topic_id",
			"title", "original_title", "content", "excerpt",
			"wp_post_id", "wp_post_url", "wp_category_ids",
			"status", "source", "is_edited", "word_count",
			"created_at", "published_at", "updated_at", "last_synced_at",
		).
		From("articles").
		Where(squirrel.Eq{"site_id": siteID}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	return r.scanArticles(query, args, ctx)
}

func (r *repository) ListByJob(ctx context.Context, jobID int64) ([]*entities.Article, error) {
	query, args := dbx.ST.
		Select(
			"id", "site_id", "job_id", "topic_id",
			"title", "original_title", "content", "excerpt",
			"wp_post_id", "wp_post_url", "wp_category_ids",
			"status", "source", "is_edited", "word_count",
			"created_at", "published_at", "updated_at", "last_synced_at",
		).
		From("articles").
		Where(squirrel.Eq{"job_id": jobID}).
		OrderBy("created_at DESC").
		MustSql()

	return r.scanArticles(query, args, ctx)
}

func (r *repository) ListByTopic(ctx context.Context, topicID int64) ([]*entities.Article, error) {
	query, args := dbx.ST.
		Select(
			"id", "site_id", "job_id", "topic_id",
			"title", "original_title", "content", "excerpt",
			"wp_post_id", "wp_post_url", "wp_category_ids",
			"status", "source", "is_edited", "word_count",
			"created_at", "published_at", "updated_at", "last_synced_at",
		).
		From("articles").
		Where(squirrel.Eq{"topic_id": topicID}).
		OrderBy("created_at DESC").
		MustSql()

	return r.scanArticles(query, args, ctx)
}

func (r *repository) GetByStatus(ctx context.Context, siteID int64, status entities.Status) ([]*entities.Article, error) {
	query, args := dbx.ST.
		Select(
			"id", "site_id", "job_id", "topic_id",
			"title", "original_title", "content", "excerpt",
			"wp_post_id", "wp_post_url", "wp_category_ids",
			"status", "source", "is_edited", "word_count",
			"created_at", "published_at", "updated_at", "last_synced_at",
		).
		From("articles").
		Where(squirrel.Eq{"site_id": siteID, "status": status}).
		OrderBy("created_at DESC").
		MustSql()

	return r.scanArticles(query, args, ctx)
}

func (r *repository) GetBySource(ctx context.Context, siteID int64, source entities.Source) ([]*entities.Article, error) {
	query, args := dbx.ST.
		Select(
			"id", "site_id", "job_id", "topic_id",
			"title", "original_title", "content", "excerpt",
			"wp_post_id", "wp_post_url", "wp_category_ids",
			"status", "source", "is_edited", "word_count",
			"created_at", "published_at", "updated_at", "last_synced_at",
		).
		From("articles").
		Where(squirrel.Eq{"site_id": siteID, "source": source}).
		OrderBy("created_at DESC").
		MustSql()

	return r.scanArticles(query, args, ctx)
}

func (r *repository) GetEdited(ctx context.Context, siteID int64) ([]*entities.Article, error) {
	query, args := dbx.ST.
		Select(
			"id", "site_id", "job_id", "topic_id",
			"title", "original_title", "content", "excerpt",
			"wp_post_id", "wp_post_url", "wp_category_ids",
			"status", "source", "is_edited", "word_count",
			"created_at", "published_at", "updated_at", "last_synced_at",
		).
		From("articles").
		Where(squirrel.Eq{"site_id": siteID, "is_edited": true}).
		OrderBy("updated_at DESC").
		MustSql()

	return r.scanArticles(query, args, ctx)
}

func (r *repository) CountBySite(ctx context.Context, siteID int64) (int, error) {
	query, args := dbx.ST.
		Select("COUNT(id)").
		From("articles").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	switch {
	case dbx.IsNoRows(err):
		return 0, nil
	case err != nil:
		return 0, errors.Database(err)
	}

	return count, nil
}

func (r *repository) CountByStatus(ctx context.Context, siteID int64, status entities.Status) (int, error) {
	query, args := dbx.ST.
		Select("COUNT(id)").
		From("articles").
		Where(squirrel.Eq{"site_id": siteID, "status": status}).
		MustSql()

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	switch {
	case dbx.IsNoRows(err):
		return 0, nil
	case err != nil:
		return 0, errors.Database(err)
	}

	return count, nil
}

func (r *repository) CountByJob(ctx context.Context, jobID int64) (int, error) {
	query, args := dbx.ST.
		Select("COUNT(id)").
		From("articles").
		Where(squirrel.Eq{"job_id": jobID}).
		MustSql()

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	switch {
	case dbx.IsNoRows(err):
		return 0, nil
	case err != nil:
		return 0, errors.Database(err)
	}

	return count, nil
}

func (r *repository) GetByWPPostIDs(ctx context.Context, siteID int64, wpPostIDs []int) ([]*entities.Article, error) {
	if len(wpPostIDs) == 0 {
		return []*entities.Article{}, nil
	}

	query, args := dbx.ST.
		Select(
			"id", "site_id", "job_id", "topic_id",
			"title", "original_title", "content", "excerpt",
			"wp_post_id", "wp_post_url", "wp_category_ids",
			"status", "source", "is_edited", "word_count",
			"created_at", "published_at", "updated_at", "last_synced_at",
		).
		From("articles").
		Where(squirrel.Eq{"site_id": siteID, "wp_post_id": wpPostIDs}).
		MustSql()

	return r.scanArticles(query, args, ctx)
}

func (r *repository) GetUnsynced(ctx context.Context, siteID int64, since time.Time) ([]*entities.Article, error) {
	query, args := dbx.ST.
		Select(
			"id", "site_id", "job_id", "topic_id",
			"title", "original_title", "content", "excerpt",
			"wp_post_id", "wp_post_url", "wp_category_ids",
			"status", "source", "is_edited", "word_count",
			"created_at", "published_at", "updated_at", "last_synced_at",
		).
		From("articles").
		Where(squirrel.Eq{"site_id": siteID}).
		Where(squirrel.Or{
			squirrel.Eq{"last_synced_at": nil},
			squirrel.LtOrEq{"last_synced_at": since},
		}).
		OrderBy("created_at DESC").
		MustSql()

	return r.scanArticles(query, args, ctx)
}

func (r *repository) UpdateSyncStatus(ctx context.Context, id int64, lastSyncedAt time.Time) error {
	query, args := dbx.ST.
		Update("articles").
		Set("last_synced_at", lastSyncedAt).
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
		return errors.NotFound("article", id)
	}

	return nil
}

func (r *repository) UpdatePublishStatus(ctx context.Context, id int64, status entities.Status, publishedAt *time.Time) error {
	builder := dbx.ST.
		Update("articles").
		Set("status", status).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id})

	if publishedAt != nil {
		builder = builder.Set("published_at", publishedAt)
	}

	query, args := builder.MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("article", id)
	}

	return nil
}

func (r *repository) BulkCreate(ctx context.Context, articles []*entities.Article) error {
	if len(articles) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Database(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, article := range articles {
		var categoryIDsJSON []byte
		categoryIDsJSON, err = json.Marshal(article.WPCategoryIDs)
		if err != nil {
			return errors.Validation("Invalid category IDs format")
		}

		query, args := dbx.ST.
			Insert("articles").
			Columns(
				"site_id", "job_id", "topic_id",
				"title", "original_title", "content", "excerpt",
				"wp_post_id", "wp_post_url", "wp_category_ids",
				"status", "source", "is_edited", "word_count",
				"published_at", "last_synced_at",
			).
			Values(
				article.SiteID, article.JobID, article.TopicID,
				article.Title, article.OriginalTitle, article.Content, article.Excerpt,
				article.WPPostID, article.WPPostURL, categoryIDsJSON,
				article.Status, article.Source, article.IsEdited, article.WordCount,
				article.PublishedAt, article.LastSyncedAt,
			).
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

func (r *repository) BulkUpdateWPInfo(ctx context.Context, updates []*entities.WPInfoUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Database(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, update := range updates {
		builder := dbx.ST.
			Update("articles").
			Set("wp_post_id", update.WPPostID).
			Set("wp_post_url", update.WPPostURL).
			Set("status", update.Status).
			Set("updated_at", time.Now()).
			Where(squirrel.Eq{"id": update.ID})

		if update.PublishedAt != nil {
			builder = builder.Set("published_at", update.PublishedAt)
		}

		query, args := builder.MustSql()

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

// Helper methods for scanning
func (r *repository) scanArticle(query string, args []interface{}, ctx context.Context) (*entities.Article, error) {
	var article entities.Article
	var jobID sql.NullInt64
	var excerpt, categoryIDsJSON sql.NullString
	var wordCount sql.NullInt32
	var publishedAt, lastSyncedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&article.ID,
		&article.SiteID,
		&jobID,
		&article.TopicID,
		&article.Title,
		&article.OriginalTitle,
		&article.Content,
		&excerpt,
		&article.WPPostID,
		&article.WPPostURL,
		&categoryIDsJSON,
		&article.Status,
		&article.Source,
		&article.IsEdited,
		&wordCount,
		&article.CreatedAt,
		&publishedAt,
		&article.UpdatedAt,
		&lastSyncedAt,
	)
	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if jobID.Valid {
		article.JobID = &jobID.Int64
	}
	if excerpt.Valid {
		article.Excerpt = &excerpt.String
	}
	if wordCount.Valid {
		count := int(wordCount.Int32)
		article.WordCount = &count
	}
	if publishedAt.Valid {
		article.PublishedAt = &publishedAt.Time
	}
	if lastSyncedAt.Valid {
		article.LastSyncedAt = &lastSyncedAt.Time
	}

	// Parse category IDs JSON
	if categoryIDsJSON.Valid {
		var categoryIDs []int
		if err = json.Unmarshal([]byte(categoryIDsJSON.String), &categoryIDs); err != nil {
			return nil, errors.Database(err)
		}
		article.WPCategoryIDs = categoryIDs
	} else {
		article.WPCategoryIDs = []int{}
	}

	return &article, nil
}

func (r *repository) scanArticles(query string, args []interface{}, ctx context.Context) ([]*entities.Article, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var articles []*entities.Article
	for rows.Next() {
		var article *entities.Article
		article, err = r.scanArticleFromRow(rows)
		if err != nil {
			return nil, errors.Database(err)
		}
		articles = append(articles, article)
	}

	switch {
	case dbx.IsNoRows(err) || len(articles) == 0:
		return articles, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return articles, nil
}

func (r *repository) scanArticleFromRow(rows *sql.Rows) (*entities.Article, error) {
	var article entities.Article
	var jobID sql.NullInt64
	var excerpt, categoryIDsJSON sql.NullString
	var wordCount sql.NullInt32
	var publishedAt, lastSyncedAt sql.NullTime

	err := rows.Scan(
		&article.ID,
		&article.SiteID,
		&jobID,
		&article.TopicID,
		&article.Title,
		&article.OriginalTitle,
		&article.Content,
		&excerpt,
		&article.WPPostID,
		&article.WPPostURL,
		&categoryIDsJSON,
		&article.Status,
		&article.Source,
		&article.IsEdited,
		&wordCount,
		&article.CreatedAt,
		&publishedAt,
		&article.UpdatedAt,
		&lastSyncedAt,
	)
	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if jobID.Valid {
		article.JobID = &jobID.Int64
	}
	if excerpt.Valid {
		article.Excerpt = &excerpt.String
	}
	if wordCount.Valid {
		count := int(wordCount.Int32)
		article.WordCount = &count
	}
	if publishedAt.Valid {
		article.PublishedAt = &publishedAt.Time
	}
	if lastSyncedAt.Valid {
		article.LastSyncedAt = &lastSyncedAt.Time
	}

	// Parse category IDs JSON
	if categoryIDsJSON.Valid {
		var categoryIDs []int
		if err = json.Unmarshal([]byte(categoryIDsJSON.String), &categoryIDs); err != nil {
			return nil, errors.Database(err)
		}
		article.WPCategoryIDs = categoryIDs
	} else {
		article.WPCategoryIDs = []int{}
	}

	return &article, nil
}
