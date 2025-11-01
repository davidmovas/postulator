package linking

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/Masterminds/squirrel"
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
			WithScope("linking_tasks"),
	}
}

func (r *repository) CreateTask(ctx context.Context, task *Task) error {
	siteIDsJSON, err := json.Marshal(task.SiteIDs)
	if err != nil {
		return errors.Validation("Invalid site IDs format")
	}

	articleIDsJSON, err := json.Marshal(task.ArticleIDs)
	if err != nil {
		return errors.Validation("Invalid article IDs format")
	}

	query, args := dbx.ST.
		Insert("linking_tasks").
		Columns(
			"name", "site_ids", "article_ids",
			"max_links_per_article", "min_link_distance",
			"prompt_id", "ai_provider_id", "status",
			"error_message", "started_at", "completed_at", "applied_at",
		).
		Values(
			task.Name, siteIDsJSON, articleIDsJSON,
			task.MaxLinksPerArticle, task.MinLinkDistance,
			task.PromptID, task.AIProviderID, task.Status,
			task.ErrorMessage, task.StartedAt, task.CompletedAt, task.AppliedAt,
		).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsForeignKeyViolation(err):
		return errors.Validation("Invalid prompt or AI provider ID")
	case err != nil:
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}

	task.ID = id
	return nil
}

func (r *repository) GetTaskByID(ctx context.Context, id int64) (*Task, error) {
	query, args := dbx.ST.
		Select(
			"id", "name", "site_ids", "article_ids",
			"max_links_per_article", "min_link_distance",
			"prompt_id", "ai_provider_id", "status",
			"error_message", "created_at", "started_at", "completed_at", "applied_at",
		).
		From("linking_tasks").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	task, err := r.scanTask(query, args, ctx)
	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("linking_task", id)
	case err != nil:
		return nil, errors.Database(err)
	}

	return task, nil
}

func (r *repository) GetAllTasks(ctx context.Context) ([]*Task, error) {
	query, args := dbx.ST.
		Select(
			"id", "name", "site_ids", "article_ids",
			"max_links_per_article", "min_link_distance",
			"prompt_id", "ai_provider_id", "status",
			"error_message", "created_at", "started_at", "completed_at", "applied_at",
		).
		From("linking_tasks").
		OrderBy("created_at DESC").
		MustSql()

	return r.scanTasks(query, args, ctx)
}

func (r *repository) GetPendingTasks(ctx context.Context) ([]*Task, error) {
	query, args := dbx.ST.
		Select(
			"id", "name", "site_ids", "article_ids",
			"max_links_per_article", "min_link_distance",
			"prompt_id", "ai_provider_id", "status",
			"error_message", "created_at", "started_at", "completed_at", "applied_at",
		).
		From("linking_tasks").
		Where(squirrel.Eq{"status": StatusPending}).
		OrderBy("created_at ASC").
		MustSql()

	return r.scanTasks(query, args, ctx)
}

func (r *repository) UpdateTask(ctx context.Context, task *Task) error {
	siteIDsJSON, err := json.Marshal(task.SiteIDs)
	if err != nil {
		return errors.Validation("Invalid site IDs format")
	}

	articleIDsJSON, err := json.Marshal(task.ArticleIDs)
	if err != nil {
		return errors.Validation("Invalid article IDs format")
	}

	query, args := dbx.ST.
		Update("linking_tasks").
		Set("name", task.Name).
		Set("site_ids", siteIDsJSON).
		Set("article_ids", articleIDsJSON).
		Set("max_links_per_article", task.MaxLinksPerArticle).
		Set("min_link_distance", task.MinLinkDistance).
		Set("prompt_id", task.PromptID).
		Set("ai_provider_id", task.AIProviderID).
		Set("status", task.Status).
		Set("error_message", task.ErrorMessage).
		Set("started_at", task.StartedAt).
		Set("completed_at", task.CompletedAt).
		Set("applied_at", task.AppliedAt).
		Where(squirrel.Eq{"id": task.ID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsForeignKeyViolation(err):
		return errors.Validation("Invalid prompt or AI provider ID")
	case err != nil:
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("linking_task", task.ID)
	}

	return nil
}

func (r *repository) DeleteTask(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("linking_tasks").
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
		return errors.NotFound("linking_task", id)
	}

	return nil
}

func (r *repository) scanTask(query string, args []any, ctx context.Context) (*Task, error) {
	var task Task
	var promptID sql.NullInt64
	var errorMessage, siteIDsJSON, articleIDsJSON sql.NullString
	var startedAt, completedAt, appliedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&task.ID,
		&task.Name,
		&siteIDsJSON,
		&articleIDsJSON,
		&task.MaxLinksPerArticle,
		&task.MinLinkDistance,
		&promptID,
		&task.AIProviderID,
		&task.Status,
		&errorMessage,
		&task.CreatedAt,
		&startedAt,
		&completedAt,
		&appliedAt,
	)

	if err != nil {
		return nil, err
	}

	if promptID.Valid {
		task.PromptID = &promptID.Int64
	}
	if errorMessage.Valid {
		task.ErrorMessage = &errorMessage.String
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	if appliedAt.Valid {
		task.AppliedAt = &appliedAt.Time
	}

	if siteIDsJSON.Valid {
		var siteIDs []int64
		if err = json.Unmarshal([]byte(siteIDsJSON.String), &siteIDs); err != nil {
			return nil, errors.Database(err)
		}
		task.SiteIDs = siteIDs
	} else {
		task.SiteIDs = []int64{}
	}

	if articleIDsJSON.Valid {
		var articleIDs []int64
		if err = json.Unmarshal([]byte(articleIDsJSON.String), &articleIDs); err != nil {
			return nil, errors.Database(err)
		}
		task.ArticleIDs = articleIDs
	} else {
		task.ArticleIDs = []int64{}
	}

	return &task, nil
}

func (r *repository) scanTasks(query string, args []any, ctx context.Context) ([]*Task, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var tasks []*Task
	for rows.Next() {
		var task *Task
		task, err = r.scanTaskFromRow(rows)
		if err != nil {
			return nil, errors.Database(err)
		}
		tasks = append(tasks, task)
	}

	switch {
	case dbx.IsNoRows(err) || len(tasks) == 0:
		return tasks, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return tasks, nil
}

func (r *repository) scanTaskFromRow(rows *sql.Rows) (*Task, error) {
	var task Task
	var promptID sql.NullInt64
	var errorMessage, siteIDsJSON, articleIDsJSON sql.NullString
	var startedAt, completedAt, appliedAt sql.NullTime

	err := rows.Scan(
		&task.ID,
		&task.Name,
		&siteIDsJSON,
		&articleIDsJSON,
		&task.MaxLinksPerArticle,
		&task.MinLinkDistance,
		&promptID,
		&task.AIProviderID,
		&task.Status,
		&errorMessage,
		&task.CreatedAt,
		&startedAt,
		&completedAt,
		&appliedAt,
	)

	if err != nil {
		return nil, err
	}

	if promptID.Valid {
		task.PromptID = &promptID.Int64
	}
	if errorMessage.Valid {
		task.ErrorMessage = &errorMessage.String
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	if appliedAt.Valid {
		task.AppliedAt = &appliedAt.Time
	}

	if siteIDsJSON.Valid {
		var siteIDs []int64
		if err = json.Unmarshal([]byte(siteIDsJSON.String), &siteIDs); err != nil {
			return nil, errors.Database(err)
		}
		task.SiteIDs = siteIDs
	} else {
		task.SiteIDs = []int64{}
	}

	if articleIDsJSON.Valid {
		var articleIDs []int64
		if err = json.Unmarshal([]byte(articleIDsJSON.String), &articleIDs); err != nil {
			return nil, errors.Database(err)
		}
		task.ArticleIDs = articleIDs
	} else {
		task.ArticleIDs = []int64{}
	}

	return &task, nil
}

func (r *linkRepository) scanLinks(query string, args []any, ctx context.Context) ([]*Link, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var links []*Link
	for rows.Next() {
		var link Link
		var targetArticleID, taskID sql.NullInt64
		var position sql.NullInt32

		err = rows.Scan(
			&link.ID,
			&link.ArticleID,
			&link.LinkType,
			&targetArticleID,
			&link.URL,
			&link.AnchorText,
			&position,
			&taskID,
			&link.CreatedAt,
		)
		if err != nil {
			return nil, errors.Database(err)
		}

		if targetArticleID.Valid {
			link.TargetArticleID = &targetArticleID.Int64
		}
		if position.Valid {
			pos := int(position.Int32)
			link.Position = &pos
		}
		if taskID.Valid {
			link.TaskID = &taskID.Int64
		}

		links = append(links, &link)
	}

	switch {
	case dbx.IsNoRows(err) || len(links) == 0:
		return links, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return links, nil
}
