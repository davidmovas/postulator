package repository

import (
	"Postulator/internal/models"
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
)

func (r *Repository) GetPrompts(ctx context.Context, limit int, offset int) (*models.PaginationResult[*models.Prompt], error) {
	query, args := builder.
		Select(
			"id",
			"name",
			"system_prompt",
			"user_prompt",
			"is_default",
			"created_at",
			"updated_at",
		).
		From("prompts").
		OrderBy("name").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query prompts: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var prompts []*models.Prompt
	for rows.Next() {
		var prompt models.Prompt
		if err = rows.Scan(
			&prompt.ID,
			&prompt.Name,
			&prompt.SystemPrompt,
			&prompt.UserPrompt,
			&prompt.IsDefault,
			&prompt.CreatedAt,
			&prompt.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan prompt: %w", err)
		}
		prompts = append(prompts, &prompt)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(id)").
		From("prompts").
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count prompts: %w", err)
	}

	return &models.PaginationResult[*models.Prompt]{
		Data:   prompts,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (r *Repository) GetPrompt(ctx context.Context, id int64) (*models.Prompt, error) {
	query, args := builder.
		Select(
			"id",
			"name",
			"system_prompt",
			"user_prompt",
			"is_default",
			"created_at",
			"updated_at",
		).
		From("prompts").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var prompt models.Prompt
	if err := r.db.QueryRowContext(ctx, query, args...).
		Scan(
			&prompt.ID,
			&prompt.Name,
			&prompt.SystemPrompt,
			&prompt.UserPrompt,
			&prompt.IsDefault,
			&prompt.CreatedAt,
			&prompt.UpdatedAt,
		); err != nil {
		return nil, fmt.Errorf("failed to query prompt: %w", err)
	}

	return &prompt, nil
}

func (r *Repository) CreatePrompt(ctx context.Context, prompt *models.Prompt) (*models.Prompt, error) {
	now := time.Now()
	prompt.CreatedAt = now
	prompt.UpdatedAt = now

	query, args := builder.
		Insert("prompts").
		Columns("name", "system_prompt", "user_prompt", "is_default", "created_at", "updated_at").
		Values(prompt.Name, prompt.SystemPrompt, prompt.UserPrompt, prompt.IsDefault, prompt.CreatedAt, prompt.UpdatedAt).
		Suffix("RETURNING id").
		MustSql()

	err := r.db.QueryRowContext(ctx, query, args...).Scan(&prompt.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt: %w", err)
	}

	return prompt, nil
}

func (r *Repository) UpdatePrompt(ctx context.Context, prompt *models.Prompt) (*models.Prompt, error) {
	prompt.UpdatedAt = time.Now()

	query, args := builder.
		Update("prompts").
		Set("name", prompt.Name).
		Set("system_prompt", prompt.SystemPrompt).
		Set("user_prompt", prompt.UserPrompt).
		Set("is_default", prompt.IsDefault).
		Set("updated_at", prompt.UpdatedAt).
		Where(squirrel.Eq{"id": prompt.ID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update prompt: %w", err)
	}

	return prompt, nil
}

func (r *Repository) DeletePrompt(ctx context.Context, id int64) error {
	query, args := builder.
		Delete("prompts").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete prompt: %w", err)
	}

	return nil
}

func (r *Repository) GetDefaultPrompt(ctx context.Context) (*models.Prompt, error) {
	query, args := builder.
		Select(
			"id",
			"name",
			"system_prompt",
			"user_prompt",
			"is_default",
			"created_at",
			"updated_at",
		).
		From("prompts").
		Where(squirrel.Eq{"is_default": true}).
		MustSql()

	var prompt models.Prompt
	if err := r.db.QueryRowContext(ctx, query, args...).
		Scan(
			&prompt.ID,
			&prompt.Name,
			&prompt.SystemPrompt,
			&prompt.UserPrompt,
			&prompt.IsDefault,
			&prompt.CreatedAt,
			&prompt.UpdatedAt,
		); err != nil {
		return nil, fmt.Errorf("failed to query default prompt: %w", err)
	}

	return &prompt, nil
}

func (r *Repository) SetDefaultPrompt(ctx context.Context, id int64) error {
	// First, unset all existing defaults
	unsetQuery, unsetArgs := builder.
		Update("prompts").
		Set("is_default", false).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"is_default": true}).
		Where(squirrel.NotEq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, unsetQuery, unsetArgs...)
	if err != nil {
		return fmt.Errorf("failed to unset existing default prompts: %w", err)
	}

	// Then set the new default
	setQuery, setArgs := builder.
		Update("prompts").
		Set("is_default", true).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err = r.db.ExecContext(ctx, setQuery, setArgs...)
	if err != nil {
		return fmt.Errorf("failed to set default prompt: %w", err)
	}

	return nil
}

func (r *Repository) CreateSitePrompt(ctx context.Context, sitePrompt *models.SitePrompt) (*models.SitePrompt, error) {
	now := time.Now()
	sitePrompt.CreatedAt = now
	sitePrompt.UpdatedAt = now

	query, args := builder.
		Insert("site_prompts").
		Columns("site_id", "prompt_id", "created_at", "updated_at").
		Values(sitePrompt.SiteID, sitePrompt.PromptID, sitePrompt.CreatedAt, sitePrompt.UpdatedAt).
		Suffix("RETURNING id").
		MustSql()

	err := r.db.QueryRowContext(ctx, query, args...).Scan(&sitePrompt.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create site prompt: %w", err)
	}

	return sitePrompt, nil
}

func (r *Repository) GetSitePrompt(ctx context.Context, siteID int64) (*models.SitePrompt, error) {
	query, args := builder.
		Select(
			"id",
			"site_id",
			"prompt_id",
			"created_at",
			"updated_at",
		).
		From("site_prompts").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	var sitePrompt models.SitePrompt
	if err := r.db.QueryRowContext(ctx, query, args...).
		Scan(
			&sitePrompt.ID,
			&sitePrompt.SiteID,
			&sitePrompt.PromptID,
			&sitePrompt.CreatedAt,
			&sitePrompt.UpdatedAt,
		); err != nil {
		return nil, fmt.Errorf("failed to query site prompt: %w", err)
	}

	return &sitePrompt, nil
}

func (r *Repository) UpdateSitePrompt(ctx context.Context, sitePrompt *models.SitePrompt) (*models.SitePrompt, error) {
	sitePrompt.UpdatedAt = time.Now()

	query, args := builder.
		Update("site_prompts").
		Set("prompt_id", sitePrompt.PromptID).
		Set("updated_at", sitePrompt.UpdatedAt).
		Where(squirrel.Eq{"id": sitePrompt.ID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update site prompt: %w", err)
	}

	return sitePrompt, nil
}

func (r *Repository) DeleteSitePrompt(ctx context.Context, id int64) error {
	query, args := builder.
		Delete("site_prompts").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete site prompt: %w", err)
	}

	return nil
}

func (r *Repository) DeleteSitePromptBySite(ctx context.Context, siteID int64) error {
	query, args := builder.
		Delete("site_prompts").
		Where(squirrel.Eq{"site_id": siteID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete site prompt by site: %w", err)
	}

	return nil
}

func (r *Repository) GetPromptSites(ctx context.Context, promptID int64, limit int, offset int) (*models.PaginationResult[*models.SitePrompt], error) {
	query, args := builder.
		Select(
			"id",
			"site_id",
			"prompt_id",
			"created_at",
			"updated_at",
		).
		From("site_prompts").
		Where(squirrel.Eq{"prompt_id": promptID}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query prompt sites: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var sitePrompts []*models.SitePrompt
	for rows.Next() {
		var sitePrompt models.SitePrompt
		if err = rows.Scan(
			&sitePrompt.ID,
			&sitePrompt.SiteID,
			&sitePrompt.PromptID,
			&sitePrompt.CreatedAt,
			&sitePrompt.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan site prompt: %w", err)
		}
		sitePrompts = append(sitePrompts, &sitePrompt)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(id)").
		From("site_prompts").
		Where(squirrel.Eq{"prompt_id": promptID}).
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count prompt sites: %w", err)
	}

	return &models.PaginationResult[*models.SitePrompt]{
		Data:   sitePrompts,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}
