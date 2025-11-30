package deletion

import (
	"context"

	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/pkg/dbx"
)

// Validator checks for dependencies before allowing entity deletion
type Validator struct {
	db *database.DB
}

// NewValidator creates a new deletion validator
func NewValidator(db *database.DB) *Validator {
	return &Validator{db: db}
}

// CanDeleteSite checks if a site can be safely deleted
// Sites are referenced by: jobs, categories, articles, site_topics, used_topics
// Note: Most have ON DELETE CASCADE, but we want to warn users about data loss
func (v *Validator) CanDeleteSite(ctx context.Context, siteID int64, siteName string) error {
	var deps []Dependency

	// Check jobs
	jobs, err := v.getJobsBySite(ctx, siteID)
	if err != nil {
		return err
	}
	deps = append(deps, jobs...)

	// Check articles
	articles, err := v.getArticlesBySite(ctx, siteID)
	if err != nil {
		return err
	}
	deps = append(deps, articles...)

	if len(deps) > 0 {
		return NewConflictError(DepSite, siteID, siteName, deps)
	}

	return nil
}

// CanDeleteTopic checks if a topic can be safely deleted
// Topics are referenced by: job_topics, site_topics, used_topics, articles, job_executions
func (v *Validator) CanDeleteTopic(ctx context.Context, topicID int64, topicTitle string) error {
	var deps []Dependency

	// Check jobs that use this topic
	jobs, err := v.getJobsByTopic(ctx, topicID)
	if err != nil {
		return err
	}
	deps = append(deps, jobs...)

	// Check articles
	articles, err := v.getArticlesByTopic(ctx, topicID)
	if err != nil {
		return err
	}
	deps = append(deps, articles...)

	if len(deps) > 0 {
		return NewConflictError(DepTopic, topicID, topicTitle, deps)
	}

	return nil
}

// CanDeletePrompt checks if a prompt can be safely deleted
// Prompts are referenced by: jobs (ON DELETE RESTRICT), job_executions (ON DELETE RESTRICT)
func (v *Validator) CanDeletePrompt(ctx context.Context, promptID int64, promptName string) error {
	var deps []Dependency

	// Check jobs
	jobs, err := v.getJobsByPrompt(ctx, promptID)
	if err != nil {
		return err
	}
	deps = append(deps, jobs...)

	if len(deps) > 0 {
		return NewConflictError(DepPrompt, promptID, promptName, deps)
	}

	return nil
}

// CanDeleteProvider checks if an AI provider can be safely deleted
// Providers are referenced by: jobs (ON DELETE RESTRICT), job_executions (ON DELETE RESTRICT)
func (v *Validator) CanDeleteProvider(ctx context.Context, providerID int64, providerName string) error {
	var deps []Dependency

	// Check jobs
	jobs, err := v.getJobsByProvider(ctx, providerID)
	if err != nil {
		return err
	}
	deps = append(deps, jobs...)

	if len(deps) > 0 {
		return NewConflictError(DepProvider, providerID, providerName, deps)
	}

	return nil
}

// CanDeleteCategory checks if a category can be safely deleted
// Categories are referenced by: job_categories, category_statistics
func (v *Validator) CanDeleteCategory(ctx context.Context, categoryID int64, categoryName string) error {
	var deps []Dependency

	// Check jobs that use this category
	jobs, err := v.getJobsByCategory(ctx, categoryID)
	if err != nil {
		return err
	}
	deps = append(deps, jobs...)

	if len(deps) > 0 {
		return NewConflictError(DepCategory, categoryID, categoryName, deps)
	}

	return nil
}

// Helper methods to query dependencies

func (v *Validator) getJobsBySite(ctx context.Context, siteID int64) ([]Dependency, error) {
	query, args := dbx.ST.
		Select("id", "name").
		From("jobs").
		Where("site_id = ?", siteID).
		Limit(10).
		MustSql()

	return v.queryDependencies(ctx, query, args, DepJob)
}

func (v *Validator) getJobsByTopic(ctx context.Context, topicID int64) ([]Dependency, error) {
	query, args := dbx.ST.
		Select("j.id", "j.name").
		From("jobs j").
		Join("job_topics jt ON j.id = jt.job_id").
		Where("jt.topic_id = ?", topicID).
		Limit(10).
		MustSql()

	return v.queryDependencies(ctx, query, args, DepJob)
}

func (v *Validator) getJobsByPrompt(ctx context.Context, promptID int64) ([]Dependency, error) {
	query, args := dbx.ST.
		Select("id", "name").
		From("jobs").
		Where("prompt_id = ?", promptID).
		Limit(10).
		MustSql()

	return v.queryDependencies(ctx, query, args, DepJob)
}

func (v *Validator) getJobsByProvider(ctx context.Context, providerID int64) ([]Dependency, error) {
	query, args := dbx.ST.
		Select("id", "name").
		From("jobs").
		Where("ai_provider_id = ?", providerID).
		Limit(10).
		MustSql()

	return v.queryDependencies(ctx, query, args, DepJob)
}

func (v *Validator) getJobsByCategory(ctx context.Context, categoryID int64) ([]Dependency, error) {
	query, args := dbx.ST.
		Select("j.id", "j.name").
		From("jobs j").
		Join("job_categories jc ON j.id = jc.job_id").
		Where("jc.category_id = ?", categoryID).
		Limit(10).
		MustSql()

	return v.queryDependencies(ctx, query, args, DepJob)
}

func (v *Validator) getArticlesBySite(ctx context.Context, siteID int64) ([]Dependency, error) {
	query, args := dbx.ST.
		Select("id", "title").
		From("articles").
		Where("site_id = ?", siteID).
		Limit(10).
		MustSql()

	return v.queryDependencies(ctx, query, args, DepArticle)
}

func (v *Validator) getArticlesByTopic(ctx context.Context, topicID int64) ([]Dependency, error) {
	query, args := dbx.ST.
		Select("id", "title").
		From("articles").
		Where("topic_id = ?", topicID).
		Limit(10).
		MustSql()

	return v.queryDependencies(ctx, query, args, DepArticle)
}

func (v *Validator) queryDependencies(ctx context.Context, query string, args []any, depType DependencyType) ([]Dependency, error) {
	rows, err := v.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []Dependency
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		deps = append(deps, Dependency{
			Type: depType,
			ID:   id,
			Name: name,
		})
	}

	return deps, rows.Err()
}
