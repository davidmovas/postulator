package repository

import (
	"Postulator/internal/models"
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
)

func (r *Repository) GetSites(ctx context.Context, limit int, offset int) (*models.PaginationResult[*models.Site], error) {
	query, args := builder.
		Select(
			"id",
			"name",
			"url",
			"username",
			"password",
			"is_active",
			"last_check",
			"status",
			"strategy",
			"created_at",
			"updated_at",
		).
		From("sites").
		OrderBy("name").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query sites: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var sites []*models.Site
	for rows.Next() {
		var site models.Site
		if err = rows.Scan(
			&site.ID,
			&site.Name,
			&site.URL,
			&site.Username,
			&site.Password,
			&site.IsActive,
			&site.LastCheck,
			&site.Status,
			&site.Strategy,
			&site.CreatedAt,
			&site.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan site: %w", err)
		}
		sites = append(sites, &site)
	}

	// Get total count
	countQuery, countArgs := builder.
		Select("COUNT(id)").
		From("sites").
		MustSql()

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count sites: %w", err)
	}

	return &models.PaginationResult[*models.Site]{
		Data:   sites,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (r *Repository) GetSite(ctx context.Context, id int64) (*models.Site, error) {
	query, args := builder.
		Select(
			"id",
			"name",
			"url",
			"username",
			"password",
			"is_active",
			"last_check",
			"status",
			"strategy",
			"created_at",
			"updated_at",
		).
		From("sites").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var site models.Site
	if err := r.db.QueryRowContext(ctx, query, args...).
		Scan(
			&site.ID,
			&site.Name,
			&site.URL,
			&site.Username,
			&site.Password,
			&site.IsActive,
			&site.LastCheck,
			&site.Status,
			&site.Strategy,
			&site.CreatedAt,
			&site.UpdatedAt,
		); err != nil {
		return nil, fmt.Errorf("failed to query site: %w", err)
	}

	return &site, nil
}

func (r *Repository) CreateSite(ctx context.Context, site *models.Site) (*models.Site, error) {
	// Set default last_check if not provided
	if site.LastCheck.IsZero() {
		site.LastCheck = time.Now()
	}

	query, args := builder.
		Insert("sites").
		Columns(
			"name",
			"url",
			"username",
			"password",
			"is_active",
			"last_check",
			"status",
			"strategy",
			"created_at",
			"updated_at",
		).
		Values(
			site.Name,
			site.URL,
			site.Username,
			site.Password,
			site.IsActive,
			site.LastCheck,
			site.Status,
			site.Strategy,
			site.CreatedAt,
			site.UpdatedAt,
		).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create site: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get insert id: %w", err)
	}

	site.ID = id
	return site, nil
}

func (r *Repository) UpdateSite(ctx context.Context, site *models.Site) (*models.Site, error) {
	query, args := builder.
		Update("sites").
		Set("name", site.Name).
		Set("url", site.URL).
		Set("username", site.Username).
		Set("password", site.Password).
		Set("is_active", site.IsActive).
		Set("status", site.Status).
		Set("strategy", site.Strategy).
		Set("updated_at", site.UpdatedAt).
		Where(squirrel.Eq{"id": site.ID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update site: %w", err)
	}

	return site, nil
}

func (r *Repository) ActivateSite(ctx context.Context, id int64) error {
	query, args := builder.
		Update("sites").
		Set("is_active", true).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to activate site: %w", err)
	}

	return nil
}

func (r *Repository) DeactivateSite(ctx context.Context, id int64) error {
	query, args := builder.
		Update("sites").
		Set("is_active", false).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to deactivate site: %w", err)
	}

	return nil
}

func (r *Repository) SetCheckStatus(ctx context.Context, id int64, checkTime time.Time, status string) error {
	query, args := builder.
		Update("sites").
		Set("last_check", checkTime).
		Set("status", status).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to set check status: %w", err)
	}

	return nil
}

func (r *Repository) DeleteSite(ctx context.Context, id int64) error {
	query, args := builder.
		Delete("sites").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete site: %w", err)
	}

	return nil
}
