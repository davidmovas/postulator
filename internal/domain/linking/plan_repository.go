package linking

import (
	"context"
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/davidmovas/postulator/pkg/errors"
)

type planRepository struct {
	db *sql.DB
}

func NewPlanRepository(db *sql.DB) PlanRepository {
	return &planRepository{db: db}
}

func (r *planRepository) Create(ctx context.Context, plan *LinkPlan) error {
	now := time.Now()
	plan.CreatedAt = now
	plan.UpdatedAt = now

	query, args, err := sq.Insert("link_plans").
		Columns("sitemap_id", "site_id", "name", "status", "provider_id", "prompt_id", "error", "created_at", "updated_at").
		Values(plan.SitemapID, plan.SiteID, plan.Name, plan.Status, plan.ProviderID, plan.PromptID, plan.Error, plan.CreatedAt, plan.UpdatedAt).
		ToSql()
	if err != nil {
		return errors.Database(err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}
	plan.ID = id

	return nil
}

func (r *planRepository) GetByID(ctx context.Context, id int64) (*LinkPlan, error) {
	query, args, err := sq.Select("id", "sitemap_id", "site_id", "name", "status", "provider_id", "prompt_id", "error", "created_at", "updated_at").
		From("link_plans").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, errors.Database(err)
	}

	plan := &LinkPlan{}
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&plan.ID, &plan.SitemapID, &plan.SiteID, &plan.Name, &plan.Status,
		&plan.ProviderID, &plan.PromptID, &plan.Error, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound("link_plan", id)
	}
	if err != nil {
		return nil, errors.Database(err)
	}

	return plan, nil
}

func (r *planRepository) GetBySitemapID(ctx context.Context, sitemapID int64) (*LinkPlan, error) {
	query, args, err := sq.Select("id", "sitemap_id", "site_id", "name", "status", "provider_id", "prompt_id", "error", "created_at", "updated_at").
		From("link_plans").
		Where(sq.Eq{"sitemap_id": sitemapID}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, errors.Database(err)
	}

	plan := &LinkPlan{}
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&plan.ID, &plan.SitemapID, &plan.SiteID, &plan.Name, &plan.Status,
		&plan.ProviderID, &plan.PromptID, &plan.Error, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Database(err)
	}

	return plan, nil
}

func (r *planRepository) GetActiveBySitemapID(ctx context.Context, sitemapID int64) (*LinkPlan, error) {
	activeStatuses := []string{
		string(PlanStatusDraft),
		string(PlanStatusSuggesting),
		string(PlanStatusReady),
		string(PlanStatusApplying),
	}

	query, args, err := sq.Select("id", "sitemap_id", "site_id", "name", "status", "provider_id", "prompt_id", "error", "created_at", "updated_at").
		From("link_plans").
		Where(sq.Eq{"sitemap_id": sitemapID}).
		Where(sq.Eq{"status": activeStatuses}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, errors.Database(err)
	}

	plan := &LinkPlan{}
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&plan.ID, &plan.SitemapID, &plan.SiteID, &plan.Name, &plan.Status,
		&plan.ProviderID, &plan.PromptID, &plan.Error, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Database(err)
	}

	return plan, nil
}

func (r *planRepository) List(ctx context.Context, siteID int64) ([]*LinkPlan, error) {
	query, args, err := sq.Select("id", "sitemap_id", "site_id", "name", "status", "provider_id", "prompt_id", "error", "created_at", "updated_at").
		From("link_plans").
		Where(sq.Eq{"site_id": siteID}).
		OrderBy("created_at DESC").
		ToSql()
	if err != nil {
		return nil, errors.Database(err)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer rows.Close()

	var plans []*LinkPlan
	for rows.Next() {
		plan := &LinkPlan{}
		if err := rows.Scan(
			&plan.ID, &plan.SitemapID, &plan.SiteID, &plan.Name, &plan.Status,
			&plan.ProviderID, &plan.PromptID, &plan.Error, &plan.CreatedAt, &plan.UpdatedAt,
		); err != nil {
			return nil, errors.Database(err)
		}
		plans = append(plans, plan)
	}

	return plans, nil
}

func (r *planRepository) Update(ctx context.Context, plan *LinkPlan) error {
	plan.UpdatedAt = time.Now()

	query, args, err := sq.Update("link_plans").
		Set("name", plan.Name).
		Set("status", plan.Status).
		Set("provider_id", plan.ProviderID).
		Set("prompt_id", plan.PromptID).
		Set("error", plan.Error).
		Set("updated_at", plan.UpdatedAt).
		Where(sq.Eq{"id": plan.ID}).
		ToSql()
	if err != nil {
		return errors.Database(err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return errors.NotFound("link_plan", plan.ID)
	}

	return nil
}

func (r *planRepository) Delete(ctx context.Context, id int64) error {
	query, args, err := sq.Delete("link_plans").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return errors.Database(err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Database(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.NotFound("link_plan", id)
	}

	return nil
}
