package prompt

import (
	"Postulator/internal/domain/entities"
	"Postulator/internal/infra/database"
	"Postulator/pkg/dbx"
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"
	"encoding/json"
	"time"

	"github.com/Masterminds/squirrel"
)

var _ IRepository = (*Repository)(nil)

type Repository struct {
	db     *database.DB
	logger *logger.Logger
}

func NewPromptRepository(c di.Container) (*Repository, error) {
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

func (r *Repository) Create(ctx context.Context, prompt *entities.Prompt) error {
	placeholdersJSON, err := json.Marshal(prompt.Placeholders)
	if err != nil {
		return errors.Internal(err)
	}

	query, args := dbx.ST.
		Insert("prompts").
		Columns("name", "system_prompt", "user_prompt", "placeholders").
		Values(prompt.Name, prompt.SystemPrompt, prompt.UserPrompt, string(placeholdersJSON)).
		MustSql()

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Internal(err)
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*entities.Prompt, error) {
	query, args := dbx.ST.
		Select("id", "name", "system_prompt", "user_prompt", "placeholders", "created_at", "updated_at").
		From("prompts").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var prompt entities.Prompt
	var placeholdersJSON string

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&prompt.ID,
		&prompt.Name,
		&prompt.SystemPrompt,
		&prompt.UserPrompt,
		&placeholdersJSON,
		&prompt.CreatedAt,
		&prompt.UpdatedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("prompt", id)
	case err != nil:
		return nil, errors.Internal(err)
	}

	if placeholdersJSON != "" {
		if err = json.Unmarshal([]byte(placeholdersJSON), &prompt.Placeholders); err != nil {
			return nil, errors.Internal(err)
		}
	}

	return &prompt, nil
}

func (r *Repository) GetAll(ctx context.Context) ([]*entities.Prompt, error) {
	query, args := dbx.ST.
		Select("id", "name", "system_prompt", "user_prompt", "placeholders", "created_at", "updated_at").
		From("prompts").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Internal(err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var prompts []*entities.Prompt
	for rows.Next() {
		var prompt entities.Prompt
		var placeholdersJSON string

		if err = rows.Scan(
			&prompt.ID,
			&prompt.Name,
			&prompt.SystemPrompt,
			&prompt.UserPrompt,
			&placeholdersJSON,
			&prompt.CreatedAt,
			&prompt.UpdatedAt,
		); err != nil {
			return nil, errors.Internal(err)
		}

		if placeholdersJSON != "" {
			if err = json.Unmarshal([]byte(placeholdersJSON), &prompt.Placeholders); err != nil {
				return nil, errors.Internal(err)
			}
		}

		prompts = append(prompts, &prompt)
	}

	return prompts, nil
}

func (r *Repository) Update(ctx context.Context, prompt *entities.Prompt) error {
	placeholdersJSON, err := json.Marshal(prompt.Placeholders)
	if err != nil {
		return errors.Internal(err)
	}

	query, args := dbx.ST.
		Update("prompts").
		Set("name", prompt.Name).
		Set("system_prompt", prompt.SystemPrompt).
		Set("user_prompt", prompt.UserPrompt).
		Set("placeholders", string(placeholdersJSON)).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": prompt.ID}).
		MustSql()

	_, err = r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsNoRows(err):
		return errors.NotFound("prompt", prompt.ID)
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("prompts").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	_, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsNoRows(err):
		return errors.NotFound("prompt", id)
	case err != nil:
		return errors.Internal(err)
	}

	return nil
}
