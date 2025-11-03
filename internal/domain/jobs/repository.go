package jobs

import (
	"context"
	"encoding/json"
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
			WithScope("jobs"),
	}
}

func (r *repository) Create(ctx context.Context, job *entities.Job) error {
	var scheduleConfigJSON, placeholdersJSON []byte
	var err error

	if job.Schedule != nil && job.Schedule.Config != nil {
		scheduleConfigJSON = job.Schedule.Config
	} else {
		scheduleConfigJSON = []byte("{}")
	}

	if job.PlaceholdersValues != nil {
		placeholdersJSON, err = json.Marshal(job.PlaceholdersValues)
		if err != nil {
			return errors.Database(err)
		}
	} else {
		placeholdersJSON = []byte("{}")
	}

	query, args := dbx.ST.
		Insert("jobs").
		Columns(
			"name", "site_id", "prompt_id", "ai_provider_id",
			"placeholders_values", "topic_strategy", "category_strategy",
			"requires_validation", "schedule_type", "schedule_config",
			"jitter_enabled", "jitter_minutes", "status",
		).
		Values(
			job.Name, job.SiteID, job.PromptID, job.AIProviderID,
			placeholdersJSON, job.TopicStrategy, job.CategoryStrategy,
			job.RequiresValidation, job.Schedule.Type, scheduleConfigJSON,
			job.JitterEnabled, job.JitterMinutes, job.Status,
		).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsForeignKeyViolation(err):
		return errors.Validation("Invalid site, prompt or AI provider ID")
	case err != nil:
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}

	job.ID = id

	stateRepo := NewStateRepository(r.db, r.logger)
	initialState := &entities.State{
		JobID:             id,
		TotalExecutions:   0,
		FailedExecutions:  0,
		LastCategoryIndex: 0,
	}
	if err = stateRepo.Update(ctx, initialState); err != nil {
		return err
	}

	if len(job.Categories) > 0 {
		if err = r.SetCategories(ctx, id, job.Categories); err != nil {
			return err
		}
	}

	if len(job.Topics) > 0 {
		if err = r.SetTopics(ctx, id, job.Topics); err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, id int64) (*entities.Job, error) {
	query, args := dbx.ST.
		Select(
			"id", "name", "site_id", "prompt_id", "ai_provider_id",
			"placeholders_values", "topic_strategy", "category_strategy",
			"requires_validation", "schedule_type", "schedule_config",
			"jitter_enabled", "jitter_minutes", "status",
			"created_at", "updated_at",
		).
		From("jobs").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	job, err := r.scanJob(query, args, ctx)
	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("job", id)
	case err != nil:
		return nil, errors.Database(err)
	}

	categories, err := r.GetCategories(ctx, id)
	if err != nil {
		return nil, err
	}
	job.Categories = categories

	topics, err := r.GetTopics(ctx, id)
	if err != nil {
		return nil, err
	}
	job.Topics = topics

	stateRepo := NewStateRepository(r.db, r.logger)
	state, err := stateRepo.Get(ctx, id)
	if err != nil && !dbx.IsNoRows(err) {
		return nil, err
	}
	job.State = state

	return job, nil
}

func (r *repository) GetAll(ctx context.Context) ([]*entities.Job, error) {
	query, args := dbx.ST.
		Select(
			"id", "name", "site_id", "prompt_id", "ai_provider_id",
			"topic_strategy", "category_strategy", "requires_validation",
			"schedule_type", "schedule_config",
			"jitter_enabled", "jitter_minutes", "status",
			"created_at", "updated_at",
		).
		From("jobs").
		OrderBy("created_at DESC").
		MustSql()

	jobs, err := r.scanJobs(query, args, ctx)
	if err != nil {
		return nil, err
	}

	stateRepo := NewStateRepository(r.db, r.logger)
	for _, job := range jobs {
		var state *entities.State
		state, err = stateRepo.Get(ctx, job.ID)
		if err != nil && !dbx.IsNoRows(err) {
			return nil, err
		}
		job.State = state
	}

	return jobs, nil
}

func (r *repository) GetActive(ctx context.Context) ([]*entities.Job, error) {
	query, args := dbx.ST.
		Select(
			"id", "name", "site_id", "prompt_id", "ai_provider_id",
			"topic_strategy", "category_strategy", "requires_validation",
			"schedule_type", "schedule_config",
			"jitter_enabled", "jitter_minutes", "status",
			"created_at", "updated_at",
		).
		From("jobs").
		Where(squirrel.Eq{"status": entities.JobStatusActive}).
		OrderBy("created_at DESC").
		MustSql()

	jobs, err := r.scanJobs(query, args, ctx)
	if err != nil {
		return nil, err
	}

	stateRepo := NewStateRepository(r.db, r.logger)
	for _, job := range jobs {
		var state *entities.State
		state, err = stateRepo.Get(ctx, job.ID)
		if err != nil && !dbx.IsNoRows(err) {
			return nil, err
		}
		job.State = state
	}

	return jobs, nil
}

func (r *repository) GetDue(ctx context.Context, before time.Time) ([]*entities.Job, error) {
	query, args := dbx.ST.
		Select(
			"j.id", "j.name", "j.site_id", "j.prompt_id", "j.ai_provider_id",
			"j.topic_strategy", "j.category_strategy", "j.requires_validation",
			"j.schedule_type", "j.schedule_config",
			"j.jitter_enabled", "j.jitter_minutes", "j.status",
			"j.created_at", "j.updated_at",
		).
		From("jobs j").
		Join("job_state js ON j.id = js.job_id").
		Where(squirrel.Eq{"j.status": entities.JobStatusActive}).
		Where(squirrel.LtOrEq{"js.next_run_at": before}).
		OrderBy("js.next_run_at ASC").
		MustSql()

	jobs, err := r.scanJobs(query, args, ctx)
	if err != nil {
		return nil, err
	}

	stateRepo := NewStateRepository(r.db, r.logger)
	for _, job := range jobs {
		var state *entities.State
		state, err = stateRepo.Get(ctx, job.ID)
		if err != nil && !dbx.IsNoRows(err) {
			return nil, err
		}
		job.State = state
	}

	return jobs, nil
}

func (r *repository) Update(ctx context.Context, job *entities.Job) error {
	var scheduleConfigJSON, placeholdersJSON []byte
	var err error

	if job.PlaceholdersValues != nil {
		placeholdersJSON, err = json.Marshal(job.PlaceholdersValues)
		if err != nil {
			return errors.Database(err)
		}
	} else {
		placeholdersJSON = []byte("{}")
	}

	if job.Schedule != nil && job.Schedule.Config != nil {
		scheduleConfigJSON = job.Schedule.Config
	} else {
		scheduleConfigJSON = []byte("{}")
	}

	query, args := dbx.ST.
		Update("jobs").
		Set("name", job.Name).
		Set("site_id", job.SiteID).
		Set("prompt_id", job.PromptID).
		Set("ai_provider_id", job.AIProviderID).
		Set("placeholders_values", placeholdersJSON).
		Set("topic_strategy", job.TopicStrategy).
		Set("category_strategy", job.CategoryStrategy).
		Set("requires_validation", job.RequiresValidation).
		Set("schedule_type", job.Schedule.Type).
		Set("schedule_config", scheduleConfigJSON).
		Set("jitter_enabled", job.JitterEnabled).
		Set("jitter_minutes", job.JitterMinutes).
		Set("status", job.Status).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": job.ID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsForeignKeyViolation(err):
		return errors.Validation("Invalid site, prompt or AI provider ID")
	case err != nil:
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("job", job.ID)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("jobs").
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
		return errors.NotFound("job", id)
	}

	return nil
}

func (r *repository) SetCategories(ctx context.Context, jobID int64, categoryIDs []int64) error {
	deleteQuery, deleteArgs := dbx.ST.
		Delete("job_categories").
		Where(squirrel.Eq{"job_id": jobID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, deleteQuery, deleteArgs...)
	if err != nil {
		return errors.Database(err)
	}

	if len(categoryIDs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Database(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for i, categoryID := range categoryIDs {
		query, args := dbx.ST.
			Insert("job_categories").
			Columns("job_id", "category_id", "order_index").
			Values(jobID, categoryID, i).
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

func (r *repository) GetCategories(ctx context.Context, jobID int64) ([]int64, error) {
	query, args := dbx.ST.
		Select("category_id").
		From("job_categories").
		Where(squirrel.Eq{"job_id": jobID}).
		OrderBy("order_index ASC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var categoryIDs []int64
	for rows.Next() {
		var categoryID int64
		err = rows.Scan(&categoryID)
		if err != nil {
			return nil, errors.Database(err)
		}
		categoryIDs = append(categoryIDs, categoryID)
	}

	switch {
	case dbx.IsNoRows(err) || len(categoryIDs) == 0:
		return categoryIDs, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return categoryIDs, nil
}

func (r *repository) SetTopics(ctx context.Context, jobID int64, topicIDs []int64) error {
	deleteQuery, deleteArgs := dbx.ST.
		Delete("job_topics").
		Where(squirrel.Eq{"job_id": jobID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, deleteQuery, deleteArgs...)
	if err != nil {
		return errors.Database(err)
	}

	if len(topicIDs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Database(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for i, topicID := range topicIDs {
		query, args := dbx.ST.
			Insert("job_topics").
			Columns("job_id", "topic_id", "order_index").
			Values(jobID, topicID, i).
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

func (r *repository) GetTopics(ctx context.Context, jobID int64) ([]int64, error) {
	query, args := dbx.ST.
		Select("topic_id").
		From("job_topics").
		Where(squirrel.Eq{"job_id": jobID}).
		OrderBy("order_index ASC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var topicIDs []int64
	for rows.Next() {
		var topicID int64
		err = rows.Scan(&topicID)
		if err != nil {
			return nil, errors.Database(err)
		}
		topicIDs = append(topicIDs, topicID)
	}

	switch {
	case dbx.IsNoRows(err) || len(topicIDs) == 0:
		return topicIDs, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return topicIDs, nil
}

func (r *repository) scanJob(query string, args []interface{}, ctx context.Context) (*entities.Job, error) {
	var job entities.Job
	var scheduleConfigJSON, placeholdersJSON []byte

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&job.ID,
		&job.Name,
		&job.SiteID,
		&job.PromptID,
		&job.AIProviderID,
		&placeholdersJSON,
		&job.TopicStrategy,
		&job.CategoryStrategy,
		&job.RequiresValidation,
		&job.Schedule.Type,
		&scheduleConfigJSON,
		&job.JitterEnabled,
		&job.JitterMinutes,
		&job.Status,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(placeholdersJSON) > 0 {
		placeholderValues := make(map[string]string)
		if err = json.Unmarshal(placeholdersJSON, &placeholderValues); err != nil {
			return nil, errors.Database(err)
		}
		job.PlaceholdersValues = placeholderValues
	}

	if len(scheduleConfigJSON) > 0 {
		job.Schedule = &entities.Schedule{
			Type:   job.Schedule.Type,
			Config: scheduleConfigJSON,
		}
	}

	return &job, nil
}

func (r *repository) scanJobs(query string, args []interface{}, ctx context.Context) ([]*entities.Job, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var jobs []*entities.Job
	for rows.Next() {
		var job entities.Job
		var scheduleConfigJSON, placeholdersJSON []byte

		err = rows.Scan(
			&job.ID,
			&job.Name,
			&job.SiteID,
			&job.PromptID,
			&job.AIProviderID,
			&placeholdersJSON,
			&job.TopicStrategy,
			&job.CategoryStrategy,
			&job.RequiresValidation,
			&job.Schedule.Type,
			&scheduleConfigJSON,
			&job.JitterEnabled,
			&job.JitterMinutes,
			&job.Status,
			&job.CreatedAt,
			&job.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Database(err)
		}

		if len(placeholdersJSON) > 0 {
			placeholderValues := make(map[string]string)
			if err = json.Unmarshal(placeholdersJSON, &placeholderValues); err != nil {
				return nil, errors.Database(err)
			}
			job.PlaceholdersValues = placeholderValues
		}

		if len(scheduleConfigJSON) > 0 {
			job.Schedule = &entities.Schedule{
				Type:   job.Schedule.Type,
				Config: scheduleConfigJSON,
			}
		}

		jobs = append(jobs, &job)
	}

	switch {
	case dbx.IsNoRows(err) || len(jobs) == 0:
		return jobs, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return jobs, nil
}
