package providers

import (
	"context"
	"time"

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
			WithScope("providers"),
	}
}

func (r *repository) Create(ctx context.Context, provider *Provider) error {
	query, args := dbx.ST.
		Insert("ai_providers").
		Columns("name", "provider", "model", "api_key", "is_active").
		Values(provider.Name, provider.Type, provider.Model, provider.APIKey, provider.IsActive).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists("provider")
	case err != nil:
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}

	provider.ID = id
	return nil
}

func (r *repository) GetByID(ctx context.Context, id int64) (*Provider, error) {
	query, args := dbx.ST.
		Select("id", "name", "provider", "model", "api_key", "is_active", "created_at", "updated_at").
		From("ai_providers").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var provider Provider
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&provider.ID,
		&provider.Name,
		&provider.Type,
		&provider.Model,
		&provider.APIKey,
		&provider.IsActive,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("provider", id)
	case err != nil:
		return nil, errors.Database(err)
	}

	return &provider, nil
}

func (r *repository) GetAll(ctx context.Context) ([]*Provider, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"name",
			"provider",
			"model",
			"api_key",
			"is_active",
			"created_at",
			"updated_at",
		).
		From("ai_providers").
		OrderBy("created_at DESC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var providers []*Provider
	for rows.Next() {
		var provider Provider
		err = rows.Scan(
			&provider.ID,
			&provider.Name,
			&provider.Type,
			&provider.Model,
			&provider.APIKey,
			&provider.IsActive,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Database(err)
		}
		providers = append(providers, &provider)
	}

	switch {
	case dbx.IsNoRows(err) || len(providers) == 0:
		return providers, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return providers, nil
}

func (r *repository) GetActive(ctx context.Context) ([]*Provider, error) {
	query, args := dbx.ST.
		Select(
			"id",
			"name",
			"provider",
			"model",
			"api_key",
			"is_active",
			"created_at",
			"updated_at",
		).
		From("ai_providers").
		Where(squirrel.Eq{"is_active": true}).
		OrderBy("created_at DESC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var providers []*Provider
	for rows.Next() {
		var provider Provider
		err = rows.Scan(
			&provider.ID,
			&provider.Name,
			&provider.Type,
			&provider.Model,
			&provider.APIKey,
			&provider.IsActive,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Database(err)
		}
		providers = append(providers, &provider)
	}

	switch {
	case dbx.IsNoRows(err) || len(providers) == 0:
		return providers, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return providers, nil
}

func (r *repository) Update(ctx context.Context, provider *Provider) error {
	query, args := dbx.ST.
		Update("ai_providers").
		Set("name", provider.Name).
		Set("provider", provider.Type).
		Set("model", provider.Model).
		Set("api_key", provider.APIKey).
		Set("is_active", provider.IsActive).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": provider.ID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists("provider")
	case err != nil:
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("provider", provider.ID)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("ai_providers").
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
		return errors.NotFound("provider", id)
	}

	return nil
}
