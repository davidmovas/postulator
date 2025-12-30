package prompts

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
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
			WithScope("prompts"),
	}
}

func (r *repository) Create(ctx context.Context, prompt *entities.Prompt) error {
	placeholdersJSON, err := json.Marshal(prompt.Placeholders)
	if err != nil {
		return errors.Validation("Invalid placeholders format")
	}

	contextConfigJSON, err := json.Marshal(prompt.ContextConfig)
	if err != nil {
		return errors.Validation("Invalid context config format")
	}

	category := prompt.Category
	if category == "" {
		category = entities.PromptCategoryPostGen
	}

	// Default to version 2 for new prompts
	version := prompt.Version
	if version == 0 {
		version = 2
	}

	query, args := dbx.ST.
		Insert("prompts").
		Columns("name", "category", "is_builtin", "system_prompt", "user_prompt", "placeholders", "instructions", "context_config", "version").
		Values(prompt.Name, category, prompt.IsBuiltin, prompt.SystemPrompt, prompt.UserPrompt, placeholdersJSON, prompt.Instructions, contextConfigJSON, version).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists("prompt")
	case err != nil:
		return errors.Database(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return errors.Database(err)
	}

	prompt.ID = id
	prompt.Version = version
	return nil
}

func (r *repository) GetByID(ctx context.Context, id int64) (*entities.Prompt, error) {
	query, args := dbx.ST.
		Select("id", "name", "category", "is_builtin", "system_prompt", "user_prompt", "placeholders", "instructions", "context_config", "version", "created_at", "updated_at").
		From("prompts").
		Where(squirrel.Eq{"id": id}).
		MustSql()

	var prompt entities.Prompt
	var category string
	var placeholdersJSON sql.NullString
	var instructions sql.NullString
	var contextConfigJSON sql.NullString
	var version sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&prompt.ID,
		&prompt.Name,
		&category,
		&prompt.IsBuiltin,
		&prompt.SystemPrompt,
		&prompt.UserPrompt,
		&placeholdersJSON,
		&instructions,
		&contextConfigJSON,
		&version,
		&prompt.CreatedAt,
		&prompt.UpdatedAt,
	)

	switch {
	case dbx.IsNoRows(err):
		return nil, errors.NotFound("prompt", id)
	case err != nil:
		return nil, errors.Database(err)
	}

	prompt.Category = entities.PromptCategory(category)
	prompt.Placeholders = parsePlaceholders(placeholdersJSON)
	prompt.Instructions = instructions.String
	prompt.ContextConfig = parseContextConfig(contextConfigJSON)
	prompt.Version = int(version.Int64)

	return &prompt, nil
}

func (r *repository) GetAll(ctx context.Context) ([]*entities.Prompt, error) {
	query, args := dbx.ST.
		Select("id", "name", "category", "is_builtin", "system_prompt", "user_prompt", "placeholders", "instructions", "context_config", "version", "created_at", "updated_at").
		From("prompts").
		OrderBy("is_builtin DESC", "created_at DESC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var prompts []*entities.Prompt
	for rows.Next() {
		var prompt entities.Prompt
		var category string
		var placeholdersJSON sql.NullString
		var instructions sql.NullString
		var contextConfigJSON sql.NullString
		var version sql.NullInt64

		err = rows.Scan(
			&prompt.ID,
			&prompt.Name,
			&category,
			&prompt.IsBuiltin,
			&prompt.SystemPrompt,
			&prompt.UserPrompt,
			&placeholdersJSON,
			&instructions,
			&contextConfigJSON,
			&version,
			&prompt.CreatedAt,
			&prompt.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Database(err)
		}

		prompt.Category = entities.PromptCategory(category)
		prompt.Placeholders = parsePlaceholders(placeholdersJSON)
		prompt.Instructions = instructions.String
		prompt.ContextConfig = parseContextConfig(contextConfigJSON)
		prompt.Version = int(version.Int64)

		prompts = append(prompts, &prompt)
	}

	switch {
	case dbx.IsNoRows(err) || len(prompts) == 0:
		return prompts, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return prompts, nil
}

func (r *repository) GetByCategory(ctx context.Context, category entities.PromptCategory) ([]*entities.Prompt, error) {
	query, args := dbx.ST.
		Select("id", "name", "category", "is_builtin", "system_prompt", "user_prompt", "placeholders", "instructions", "context_config", "version", "created_at", "updated_at").
		From("prompts").
		Where(squirrel.Eq{"category": string(category)}).
		OrderBy("is_builtin DESC", "created_at DESC").
		MustSql()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Database(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var prompts []*entities.Prompt
	for rows.Next() {
		var prompt entities.Prompt
		var cat string
		var placeholdersJSON sql.NullString
		var instructions sql.NullString
		var contextConfigJSON sql.NullString
		var version sql.NullInt64

		err = rows.Scan(
			&prompt.ID,
			&prompt.Name,
			&cat,
			&prompt.IsBuiltin,
			&prompt.SystemPrompt,
			&prompt.UserPrompt,
			&placeholdersJSON,
			&instructions,
			&contextConfigJSON,
			&version,
			&prompt.CreatedAt,
			&prompt.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Database(err)
		}

		prompt.Category = entities.PromptCategory(cat)
		prompt.Placeholders = parsePlaceholders(placeholdersJSON)
		prompt.Instructions = instructions.String
		prompt.ContextConfig = parseContextConfig(contextConfigJSON)
		prompt.Version = int(version.Int64)

		prompts = append(prompts, &prompt)
	}

	switch {
	case dbx.IsNoRows(err) || len(prompts) == 0:
		return prompts, nil
	case err != nil || rows.Err() != nil:
		return nil, errors.Database(err)
	}

	return prompts, nil
}

func (r *repository) Update(ctx context.Context, prompt *entities.Prompt) error {
	placeholdersJSON, err := json.Marshal(prompt.Placeholders)
	if err != nil {
		return errors.Validation("Invalid placeholders format")
	}

	contextConfigJSON, err := json.Marshal(prompt.ContextConfig)
	if err != nil {
		return errors.Validation("Invalid context config format")
	}

	category := prompt.Category
	if category == "" {
		category = entities.PromptCategoryPostGen
	}

	query, args := dbx.ST.
		Update("prompts").
		Set("name", prompt.Name).
		Set("category", string(category)).
		Set("system_prompt", prompt.SystemPrompt).
		Set("user_prompt", prompt.UserPrompt).
		Set("placeholders", placeholdersJSON).
		Set("instructions", prompt.Instructions).
		Set("context_config", contextConfigJSON).
		Set("version", prompt.Version).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": prompt.ID}).
		MustSql()

	result, err := r.db.ExecContext(ctx, query, args...)
	switch {
	case dbx.IsUniqueViolation(err):
		return errors.AlreadyExists("prompt")
	case err != nil:
		return errors.Database(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Database(err)
	}

	if rowsAffected == 0 {
		return errors.NotFound("prompt", prompt.ID)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	query, args := dbx.ST.
		Delete("prompts").
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
		return errors.NotFound("prompt", id)
	}

	return nil
}

// parsePlaceholders handles both JSON array format and comma-separated format
func parsePlaceholders(placeholdersJSON sql.NullString) []string {
	if !placeholdersJSON.Valid || strings.TrimSpace(placeholdersJSON.String) == "" {
		return []string{}
	}

	raw := strings.TrimSpace(placeholdersJSON.String)

	// Try JSON array format first
	if strings.HasPrefix(raw, "[") {
		var placeholders []string
		if err := json.Unmarshal([]byte(raw), &placeholders); err == nil {
			return placeholders
		}
	}

	// Fall back to comma-separated format
	parts := strings.Split(raw, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// parseContextConfig parses JSON context config from database
func parseContextConfig(configJSON sql.NullString) entities.ContextConfig {
	if !configJSON.Valid || strings.TrimSpace(configJSON.String) == "" {
		return nil
	}

	var config entities.ContextConfig
	if err := json.Unmarshal([]byte(configJSON.String), &config); err != nil {
		return nil
	}
	return config
}
