package aiprovider

import (
	"Postulator/internal/domain/entities"
	"Postulator/internal/infra/database"
	"Postulator/pkg/dbx"
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"
	"time"

	"github.com/Masterminds/squirrel"
)

var _ IRepository = (*Repository)(nil)

type Repository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewRepository(c di.Container) (*Repository, error) {
	var db *database.DB
	if err := c.Resolve(&db); err != nil {
		return nil, err
	}

	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	return &Repository{
		db:     db,
		logger: l,
	}, nil
}

func (r *Repository) Create(ctx context.Context, provider *entities.AIProvider) error {
	query, args := dbx.ST.
		Insert("ai_providers").
		Columns("name", "api_key", "model", "is_active").
		Values(provider.Name, provider.APIKey, provider.Model, provider.IsActive).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists("AI provider with name: " + provider.Name)
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*entities.AIProvider, error) {
	query, args := dbx.ST.
		Select("id", "name", "api_key", "model", "is_active", "created_at", "updated_at").
		From("ai_providers").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var provider entities.AIProvider
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&provider.ID,
		&provider.Name,
		&provider.APIKey,
		&provider.Model,
		&provider.IsActive,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("AI provider", id)
	case err != nil:
		return nil, errors.Internal(err)
	}

	return &provider, nil
}

func (r *Repository) GetAll(ctx context.Context) ([]*entities.AIProvider, error) {
	query, args := dbx.ST.
		Select("id", "name", "api_key", "model", "is_active", "created_at", "updated_at").
		From("ai_providers").
		OrderBy("created_at DESC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Internal(err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var providers []*entities.AIProvider
	for rows.Next() {
		var provider entities.AIProvider
		if err = rows.Scan(
			&provider.ID,
			&provider.Name,
			&provider.APIKey,
			&provider.Model,
			&provider.IsActive,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		); err != nil {
			return nil, errors.Internal(err)
		}

		providers = append(providers, &provider)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Internal(err)
	}

	return providers, nil
}

func (r *Repository) GetActive(ctx context.Context) ([]*entities.AIProvider, error) {
	query, args := dbx.ST.
		Select("id", "name", "api_key", "model", "is_active", "created_at", "updated_at").
		From("ai_providers").
		Where(squirrel.Eq{"is_active": true}).
		OrderBy("created_at DESC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Internal(err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var providers []*entities.AIProvider
	for rows.Next() {
		var provider entities.AIProvider
		if err = rows.Scan(
			&provider.ID,
			&provider.Name,
			&provider.APIKey,
			&provider.Model,
			&provider.IsActive,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		); err != nil {
			return nil, errors.Internal(err)
		}

		providers = append(providers, &provider)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Internal(err)
	}

	return providers, nil
}

func (r *Repository) Update(ctx context.Context, provider *entities.AIProvider) error {
	query, args := dbx.ST.
		Update("ai_providers").
		Set("name", provider.Name).
		Set("api_key", provider.APIKey).
		Set("model", provider.Model).
		Set("is_active", provider.IsActive).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": provider.ID}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsNoRows(err):
		return errors.NotFound("AI provider", provider.ID)
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists("AI provider with name: " + provider.Name)
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("ai_providers").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Internal(err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return errors.Internal(err)
	}

	if affected == 0 {
		return errors.NotFound("AI provider", id)
	}

	return nil
}
